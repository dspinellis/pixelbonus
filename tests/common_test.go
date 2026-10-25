package tests

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pixelbonus/src/auth"
	"pixelbonus/src/courses"
	"pixelbonus/src/csrf"
	"pixelbonus/src/database"
	"pixelbonus/src/grades"
	"pixelbonus/src/i18n"
	"pixelbonus/src/public"
	"pixelbonus/src/redemption"
	"pixelbonus/src/session"
	"pixelbonus/src/tokens"
	"pixelbonus/src/web"

	"golang.org/x/crypto/bcrypt"
)

type fakeMailer struct {
	to      string
	subject string
	body    string
}

func (f *fakeMailer) Send(ctx context.Context, to string, subject string, body string) error {
	f.to, f.subject, f.body = to, subject, body
	return nil
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	var db *sql.DB
	var err error
	if dsn := os.Getenv("PIXELBONUS_TEST_MYSQL_DSN"); dsn != "" {
		cfg, parseErr := mysql.ParseDSN(dsn)
		if parseErr != nil {
			t.Fatal("invalid test MySQL DSN")
		}
		cfg.DBName = fmt.Sprintf("pixelbonus_test_%d", time.Now().UnixNano())
		schema, readErr := ioutil.ReadFile("../design/schema.sql")
		if readErr != nil {
			t.Fatal(readErr)
		}
		db, err = database.OpenMySQL(cfg.FormatDSN(), string(schema), true)
		if err == nil {
			t.Cleanup(func() { db.Exec("DROP DATABASE `" + cfg.DBName + "`"); db.Close() })
		}
	} else {
		schema, readErr := ioutil.ReadFile("sqlite-schema.sql")
		if readErr != nil {
			t.Fatal(readErr)
		}
		db, err = sql.Open("sqlite3", filepath.Join(t.TempDir(), "test.db")+"?_foreign_keys=on")
		if err == nil {
			_, err = db.Exec(string(schema))
		}
		if err == nil {
			t.Cleanup(func() { db.Close() })
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func newTestServer(t *testing.T) (*httptest.Server, *sql.DB, *fakeMailer) {
	t.Helper()
	db := newTestDB(t)
	mailer := &fakeMailer{}
	catalogs, err := i18n.LoadCatalogs(filepath.Join("..", "src", "translations"))
	if err != nil {
		t.Fatal(err)
	}
	courseSvc := &courses.Service{DB: db, Now: func() time.Time { return time.Unix(1000, 0) }}
	server, err := web.NewServer(web.Dependencies{
		TemplateDir: filepath.Join("..", "src", "templates"),
		Catalogs:    catalogs,
		Auth:        &auth.Service{DB: db, Mailer: mailer, BaseURL: "https://pixelbonus.com", Now: func() time.Time { return time.Unix(1000, 0) }},
		Sessions:    &session.Store{DB: db, Now: func() time.Time { return time.Unix(1000, 0) }},
		CSRF:        &csrf.Service{Secret: []byte("01234567890123456789012345678901"), Now: func() time.Time { return time.Unix(1000, 0) }, MaxAge: time.Hour},
		Courses:     courseSvc,
		Tokens:      &tokens.Service{DB: db, Courses: courseSvc, Mailer: mailer, BaseURL: "https://pixelbonus.com", Now: func() time.Time { return time.Unix(1000, 0) }},
		Grades:      &grades.Service{DB: db, Courses: courseSvc},
		Public:      &public.Service{DB: db},
		Redemption:  &redemption.Service{DB: db},
		DB:          db,
	})
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(server.Routes()), db, mailer
}

func upsertTestInstructor(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	res, err := db.Exec("INSERT INTO Users(username,username_canonical,email,email_canonical,enabled,salt,password,locked,expired,roles,credentials_expired,name,grade_multiplier,max_grade,min_grade,locale) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", "teacher@example.com", "teacher@example.com", "teacher@example.com", "teacher@example.com", true, "", string(hash), false, false, `a:0:{}`, false, "Teacher", 1, 10, 0, "en")
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}

func form(body string) *strings.Reader { return strings.NewReader(body) }

func mustWriteFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}

func testContext() context.Context { return context.Background() }

func seedCoupon(t *testing.T, db *sql.DB, courseID int64, codes ...string) {
	t.Helper()
	stamp := database.DateTime(time.Unix(1000, 0))
	res, err := db.Exec("INSERT INTO QrSet(user_id,created_at,updated_at) VALUES(?,?,?)", courseID, stamp, stamp)
	if err != nil {
		t.Fatal(err)
	}
	setID, _ := res.LastInsertId()
	for _, code := range codes {
		if _, err := db.Exec("INSERT INTO QrCode(code,qrset_id,created_at,updated_at) VALUES(?,?,?,?)", code, setID, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
}
