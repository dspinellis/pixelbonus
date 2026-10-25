package tests

import (
	"testing"
	"time"

	"pixelbonus/src/courses"
	"pixelbonus/src/redemption"
)

func TestRedeemOnce(t *testing.T) {
	db := newTestDB(t)
	instructorID := upsertTestInstructor(t, db)
	courseSvc := &courses.Service{DB: db, Now: func() time.Time { return time.Unix(1000, 0) }}
	course, err := courseSvc.CreateCourse(testContext(), instructorID, courses.CreateCourseInput{Name: "SE 2026", GradingMethod: courses.ValidGradingMethods[0]})
	if err != nil {
		t.Fatal(err)
	}
	seedCoupon(t, db, course.ID, "00112233445566778899AABBCCDDEEFF00112233")

	svc := &redemption.Service{DB: db}
	res, err := svc.Redeem(testContext(), redemption.RedeemInput{Code: "0011 2233 4455 6677 8899 AABB CCDD EEFF 0011 2233", StudentID: "s1"}, time.Unix(1001, 0))
	if err != nil || res.Status != "success" {
		t.Fatalf("unexpected redeem result: %+v %v", res, err)
	}
	res, err = svc.Redeem(testContext(), redemption.RedeemInput{Code: "00112233445566778899AABBCCDDEEFF00112233", StudentID: "s2"}, time.Unix(1002, 0))
	if err != nil || res.Status != "already" {
		t.Fatalf("unexpected second redeem result: %+v %v", res, err)
	}
}
