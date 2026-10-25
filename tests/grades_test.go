package tests

import (
	"testing"
	"time"

	"pixelbonus/src/courses"
	"pixelbonus/src/grades"
	"pixelbonus/src/public"
)

func TestComputeGrades(t *testing.T) {
	db := newTestDB(t)
	instructorID := upsertTestInstructor(t, db)
	courseSvc := &courses.Service{DB: db, Now: func() time.Time { return time.Unix(1000, 0) }}
	course, err := courseSvc.CreateCourse(testContext(), instructorID, courses.CreateCourseInput{Name: "SE 2026", GradingMethod: courses.ValidGradingMethods[0]})
	if err != nil {
		t.Fatal(err)
	}
	seedCoupon(t, db, course.ID, "00112233445566778899AABBCCDDEEFF00112233", "10112233445566778899AABBCCDDEEFF00112233", "20112233445566778899AABBCCDDEEFF00112233")
	if _, err := db.Exec("INSERT INTO Redemption(qrcode_code,participantNumber,created_at,updated_at) VALUES(?,?,?,?)", "00112233445566778899AABBCCDDEEFF00112233", "z1", "1970-01-01 00:16:41", "1970-01-01 00:16:41"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO Redemption(qrcode_code,participantNumber,created_at,updated_at) VALUES(?,?,?,?)", "10112233445566778899AABBCCDDEEFF00112233", "z1", "1970-01-01 00:16:41", "1970-01-01 00:16:41"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO Redemption(qrcode_code,participantNumber,created_at,updated_at) VALUES(?,?,?,?)", "20112233445566778899AABBCCDDEEFF00112233", "b1", "1970-01-01 00:16:41", "1970-01-01 00:16:41"); err != nil {
		t.Fatal(err)
	}

	svc := &grades.Service{DB: db, Courses: courseSvc}
	result, err := svc.ComputeCourseGrades(testContext(), instructorID, course.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 2 || result.Rows[0].StudentID != "z1" || result.Rows[0].Redemptions != 2 || result.Rows[1].StudentID != "b1" || result.Rows[1].Redemptions != 1 {
		t.Fatalf("unexpected grades: %+v", result.Rows)
	}
	publicView, err := (&public.Service{DB: db}).ListPublicRedemptions(testContext(), course.PublicNonce)
	if err != nil {
		t.Fatal(err)
	}
	if len(publicView.Rows) != 2 || publicView.Rows[0].StudentID != "b1" || publicView.Rows[1].StudentID != "z1" {
		t.Fatalf("public rows should be sorted by student ID: %+v", publicView.Rows)
	}
}
