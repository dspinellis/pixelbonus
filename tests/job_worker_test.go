package tests

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"pixelbonus/src/pixelbonus/coupon"
	"pixelbonus/src/pixelbonus/course"
	"pixelbonus/src/pixelbonus/email"
	"pixelbonus/src/pixelbonus/job"
)

func TestWorkerCompletesQueuedJob(t *testing.T) {
	cfg := testConfig(t)
	db := testDB(t, cfg)
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO instructors (id, full_name, email, password_hash, verified_at, created_at) VALUES (1, 'A', 'a@example.com', 'x', ?, ?)`, fixedTime(), fixedTime()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO courses (id, instructor_id, name, grading_method, created_at, updated_at) VALUES (1, 1, 'Course', 'zero_to_q2_10', ?, ?)`, fixedTime(), fixedTime()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO coupon_generation_jobs (id, instructor_id, course_id, quantity, status, created_at) VALUES (1, 1, 1, 20, 'queued', ?)`, fixedTime()); err != nil {
		t.Fatal(err)
	}
	mailOut := filepath.Join(t.TempDir(), "mail.txt")
	worker := job.NewWorker(
		coupon.NewService(db, fixedTime, randomReader('c')),
		course.NewService(db, fixedTime),
		email.NewSender(writeMailScript(t, mailOut), "no-reply@pixelbonus.com"),
		cfg,
		fixedTime,
		randomReader('d'),
	)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(done)
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done
	var status string
	if err := db.QueryRow(`SELECT status FROM coupon_generation_jobs WHERE id = 1`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "completed" {
		t.Fatalf("unexpected status: %s", status)
	}
	files, err := ioutil.ReadDir(cfg.PDFDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("expected generated pdf")
	}
}
