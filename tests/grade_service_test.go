package tests

import (
	"context"
	"database/sql"
	"testing"

	"pixelbonus/src/pixelbonus/grade"
)

func TestCourseGrades(t *testing.T) {
	cfg := testConfig(t)
	db := testDB(t, cfg)
	defer db.Close()
	seedGradeData(t, db)
	svc := grade.NewService(db)
	_, rows, err := svc.CourseGrades(context.Background(), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].StudentID != "s1" {
		t.Fatalf("unexpected first student: %s", rows[0].StudentID)
	}
}

func seedGradeData(t *testing.T, db *sql.DB) {
	t.Helper()
	seedCourseData(t, db)
	if _, err := db.Exec(`INSERT INTO coupons (coupon_set_id, course_id, raw_code, normalized_code, redeemed_by_student_id, redeemed_at) VALUES (1, 1, X'BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB', 'BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB', 's1', ?)`, fixedTime()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO coupons (coupon_set_id, course_id, raw_code, normalized_code, redeemed_by_student_id, redeemed_at) VALUES (1, 1, X'CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC', 'CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC', 's1', ?)`, fixedTime()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO coupons (coupon_set_id, course_id, raw_code, normalized_code, redeemed_by_student_id, redeemed_at) VALUES (1, 1, X'DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD', 'DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD', 's2', ?)`, fixedTime()); err != nil {
		t.Fatal(err)
	}
}
