package tests

import (
	"testing"
	"time"

	"pixelbonus/src/session"
)

func TestSessionLifecycle(t *testing.T) {
	db := newTestDB(t)
	instructorID := upsertTestInstructor(t, db)
	store := &session.Store{DB: db, Now: func() time.Time { return time.Unix(1000, 0) }}
	sess, err := store.Create(testContext(), instructorID, time.Unix(1000, 0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Validate(testContext(), sess.ID, time.Unix(1001, 0)); err != nil {
		t.Fatal(err)
	}
	if err := store.Destroy(testContext(), sess.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Validate(testContext(), sess.ID, time.Unix(1002, 0)); err == nil {
		t.Fatal("expected invalid session after destroy")
	}
}
