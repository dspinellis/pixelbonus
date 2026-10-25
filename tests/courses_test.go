package tests

import (
	"testing"
	"time"

	"pixelbonus/src/courses"
)

func TestCourseCreateAndList(t *testing.T) {
	db := newTestDB(t)
	instructorID := upsertTestInstructor(t, db)
	svc := &courses.Service{DB: db, Now: func() time.Time { return time.Unix(1000, 0) }}
	created, err := svc.CreateCourse(testContext(), instructorID, courses.CreateCourseInput{Name: "SE 2026", GradingMethod: courses.ValidGradingMethods[0]})
	if err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListCourses(testContext(), instructorID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("unexpected course list: %+v", list)
	}
}
