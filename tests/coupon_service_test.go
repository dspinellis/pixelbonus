package tests

import (
	"context"
	"database/sql"
	"testing"

	"pixelbonus/src/pixelbonus/coupon"
)

func TestRedeemCoupon(t *testing.T) {
	cfg := testConfig(t)
	db := testDB(t, cfg)
	defer db.Close()
	seedCourseData(t, db)
	svc := coupon.NewService(db, fixedTime, randomReader('b'))
	result, err := svc.RedeemByCode(context.Background(), "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "s1")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "redeemed" {
		t.Fatalf("unexpected status: %s", result.Status)
	}
}

func seedCourseData(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO instructors (id, full_name, email, password_hash, verified_at, created_at) VALUES (1, 'A', 'a@example.com', 'x', ?, ?)`, fixedTime(), fixedTime()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO courses (id, instructor_id, name, grading_method, created_at, updated_at) VALUES (1, 1, 'Course', 'zero_to_q2_10', ?, ?)`, fixedTime(), fixedTime()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO coupon_sets (id, course_id, job_id, created_at) VALUES (1, 1, 1, ?)`, fixedTime()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO coupon_generation_jobs (id, instructor_id, course_id, quantity, status, created_at) VALUES (1, 1, 1, 20, 'completed', ?)`, fixedTime()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO coupons (coupon_set_id, course_id, raw_code, normalized_code) VALUES (1, 1, X'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA', 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA')`); err != nil {
		t.Fatal(err)
	}
}
