package tests

import (
	"context"
	"testing"

	"pixelbonus/src/pixelbonus/course"
)

func TestCreateAndListCourses(t *testing.T) {
	cfg := testConfig(t)
	db := testDB(t, cfg)
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO instructors (id, full_name, email, password_hash, verified_at, created_at) VALUES (1, 'A', 'a@example.com', 'x', ?, ?)`, fixedTime(), fixedTime()); err != nil {
		t.Fatal(err)
	}
	svc := course.NewService(db, fixedTime)
	if _, err := svc.CreateCourse(context.Background(), 1, course.CreateCourseInput{Name: "Course 2026", GradingMethod: "zero_to_q2_10"}); err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListCourses(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 course, got %d", len(list))
	}
}
