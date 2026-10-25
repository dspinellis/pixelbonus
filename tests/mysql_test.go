package tests

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"pixelbonus/src/auth"
	"pixelbonus/src/courses"
	"pixelbonus/src/database"
	"pixelbonus/src/grades"
	"pixelbonus/src/pdf"
	"pixelbonus/src/public"
	"pixelbonus/src/redemption"
	"pixelbonus/src/tokens"
)

func mysqlTestDB(t *testing.T, schema string) (*sql.DB, string) {
	t.Helper()
	dsn := os.Getenv("PIXELBONUS_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set PIXELBONUS_TEST_MYSQL_DSN to run real MySQL compatibility tests")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid test MySQL DSN")
	}
	cfg.DBName = fmt.Sprintf("pixelbonus_test_%d", time.Now().UnixNano())
	db, err := database.OpenMySQL(cfg.FormatDSN(), schema, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec("DROP DATABASE `" + cfg.DBName + "`"); db.Close() })
	return db, cfg.FormatDSN()
}

func TestMySQLLegacyCompatibility(t *testing.T) {
	dump, err := ioutil.ReadFile("../design/legacy-schema.md")
	if err != nil {
		t.Fatal(err)
	}
	var legacy strings.Builder
	// Import the actual legacy definitions, in foreign-key dependency order.
	for _, table := range []string{"Users", "Course", "QrSet", "QrRequest", "QrCode", "Redemption", "Tag", "qrset_tags"} {
		ddl := regexp.MustCompile("(?s)CREATE TABLE `" + table + "` .*?;").Find(dump)
		if ddl == nil {
			t.Fatalf("missing legacy table %s", table)
		}
		legacy.Write(ddl)
	}
	db, dsn := mysqlTestDB(t, legacy.String())
	instructorID := upsertTestInstructor(t, db)
	if _, err := db.Exec("UPDATE Users SET surname = 'Legacy', preferred_grading_model = ? WHERE id = ?", courses.ValidGradingMethods[2], instructorID); err != nil {
		t.Fatal(err)
	}
	stamp := "2018-01-02 03:04:05"
	res, err := db.Exec("INSERT INTO Course(user_id,name,created_at,updated_at,hashedUrl) VALUES(?,?,?,?,?)", instructorID, "Legacy course", stamp, stamp, "unchanged-public-url")
	if err != nil {
		t.Fatal(err)
	}
	courseID, _ := res.LastInsertId()
	code := "00112233445566778899AABBCCDDEEFF00112233"
	used := "10112233445566778899AABBCCDDEEFF00112233"
	seedCoupon(t, db, courseID, code, used)
	if _, err := db.Exec("INSERT INTO Redemption(participantNumber,created_at,updated_at,qrcode_code) VALUES(?,?,?,?)", "old-student", stamp, stamp, used); err != nil {
		t.Fatal(err)
	}
	schema, err := ioutil.ReadFile("../design/schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		reopened, err := database.OpenMySQL(dsn, string(schema), false)
		if err != nil {
			t.Fatal(err)
		}
		reopened.Close()
	}
	res, err = db.Exec("INSERT INTO QrRequest(qrset_id,status,quantity,created_at,updated_at) SELECT qrset_id,'complete',1,created_at,updated_at FROM QrCode WHERE code = ?", used)
	if err != nil {
		t.Fatal(err)
	}
	requestID, _ := res.LastInsertId()
	if _, err := db.Exec("UPDATE QrCode SET qrset_id = NULL, qrrequest_id = ? WHERE code = ?", requestID, used); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	authSvc := &auth.Service{DB: db}
	ident, err := authSvc.Login(ctx, "teacher@example.com", "password123")
	if err != nil || ident.ID != instructorID || ident.FullName != "Teacher Legacy" {
		t.Fatalf("legacy login: %+v %v", ident, err)
	}
	courseSvc := &courses.Service{DB: db}
	course, err := courseSvc.GetCourse(ctx, instructorID, courseID)
	if err != nil || course.PublicNonce != "unchanged-public-url" || course.GradingMethod != courses.ValidGradingMethods[2] || course.CreatedAt != stamp {
		t.Fatalf("legacy course: %+v %v", course, err)
	}
	publicSvc := &public.Service{DB: db}
	view, err := publicSvc.ListPublicRedemptions(ctx, course.PublicNonce)
	if err != nil || len(view.Rows) != 1 || view.Rows[0].StudentID != "old-student" {
		t.Fatalf("legacy redemptions: %+v %v", view, err)
	}
	gradesSvc := &grades.Service{DB: db, Courses: courseSvc}
	result, err := gradesSvc.ComputeCourseGrades(ctx, instructorID, courseID)
	if err != nil || len(result.Rows) != 1 || result.Rows[0].Redemptions != 1 {
		t.Fatalf("legacy grades: %+v %v", result, err)
	}
	redeemSvc := &redemption.Service{DB: db}
	already, err := redeemSvc.Redeem(ctx, redemption.RedeemInput{Code: used, StudentID: "another"}, time.Now())
	if err != nil || already.Status != "already" {
		t.Fatalf("legacy spent coupon: %+v %v", already, err)
	}
	// All contenders must see the same row lock; the dump has no unique key on Redemption.qrcode_code.
	const n = 8
	outcomes := make(chan string, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := redeemSvc.Redeem(ctx, redemption.RedeemInput{Code: code, StudentID: "new-student"}, time.Now())
			if err != nil {
				outcomes <- err.Error()
				return
			}
			outcomes <- result.Status
		}()
	}
	wg.Wait()
	close(outcomes)
	successes := 0
	for outcome := range outcomes {
		if outcome == "success" {
			successes++
		} else if outcome != "already" {
			t.Errorf("concurrent redemption: %s", outcome)
		}
	}
	if successes != 1 {
		t.Fatalf("wanted one redemption, got %d", successes)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM Redemption WHERE qrcode_code = ?", code).Scan(&count); err != nil || count != 1 {
		t.Fatalf("redemption count: %d %v", count, err)
	}
	if _, err := courseSvc.UpdateCourse(ctx, instructorID, courses.UpdateCourseInput{ID: courseID, Name: "Updated", GradingMethod: courses.ValidGradingMethods[1]}); err != nil {
		t.Fatal(err)
	}
	var nonce string
	if err := db.QueryRow("SELECT hashedUrl FROM Course WHERE id = ?", courseID).Scan(&nonce); err != nil || nonce != "unchanged-public-url" {
		t.Fatalf("changed public URL: %s %v", nonce, err)
	}
}

type capturePDF struct{ codes chan []pdf.CouponPrintData }

func (p *capturePDF) GenerateCoupons(ctx context.Context, path, prefix string, coupons []pdf.CouponPrintData) error {
	p.codes <- coupons
	return nil
}

func TestTokenSetLegacyMapping(t *testing.T) {
	db := newTestDB(t)
	instructorID := upsertTestInstructor(t, db)
	courseSvc := &courses.Service{DB: db}
	course, err := courseSvc.CreateCourse(testContext(), instructorID, courses.CreateCourseInput{Name: "Coupons", GradingMethod: courses.ValidGradingMethods[0]})
	if err != nil {
		t.Fatal(err)
	}
	pdfGen := &capturePDF{codes: make(chan []pdf.CouponPrintData, 1)}
	svc := &tokens.Service{DB: db, Courses: courseSvc, PDF: pdfGen, BaseURL: "https://example.com"}
	set, err := svc.CreateTokenSet(testContext(), instructorID, course.ID, 20, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	select {
	case codes := <-pdfGen.codes:
		if len(codes) != 20 {
			t.Fatalf("got %d coupons", len(codes))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("PDF job did not run")
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		path, name, err := svc.GetDownloadPath(testContext(), set.FileNonce)
		if err == nil {
			if path != set.FilePath || name != course.Name {
				t.Fatalf("wrong download: %s %s", path, name)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM QrCode q JOIN QrSet s ON s.id=q.qrset_id WHERE q.qrrequest_id=? AND s.user_id=?", set.ID, course.ID).Scan(&count); err != nil || count != 20 {
		t.Fatalf("legacy coupon mapping: %d %v", count, err)
	}
}

func TestTokenSetRollsBackOnCodeCollision(t *testing.T) {
	db := newTestDB(t)
	instructorID := upsertTestInstructor(t, db)
	courseSvc := &courses.Service{DB: db}
	course, err := courseSvc.CreateCourse(testContext(), instructorID, courses.CreateCourseInput{Name: "Rollback", GradingMethod: courses.ValidGradingMethods[0]})
	if err != nil {
		t.Fatal(err)
	}
	svc := &tokens.Service{DB: db, Courses: courseSvc, Random: func(b []byte) (int, error) { return len(b), nil }}
	if _, err := svc.CreateTokenSet(testContext(), instructorID, course.ID, 20, time.Now()); err == nil {
		t.Fatal("expected duplicate coupon failure")
	}
	for _, table := range []string{"QrSet", "QrRequest", "QrCode", "pixelbonus_pdf_jobs"} {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("partial writes in %s: %d %v", table, count, err)
		}
	}
}
