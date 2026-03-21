package tests

import (
	"context"
	"path/filepath"
	"testing"

	"pixelbonus/src/pixelbonus/auth"
	"pixelbonus/src/pixelbonus/email"
)

func TestRegisterVerifyAndLogin(t *testing.T) {
	cfg := testConfig(t)
	db := testDB(t, cfg)
	defer db.Close()
	mailOut := filepath.Join(t.TempDir(), "mail.txt")
	sender := email.NewSender(writeMailScript(t, mailOut), "no-reply@pixelbonus.com")
	svc := auth.NewService(db, sender, cfg.BaseURL, fixedTime, randomReader('a'))

	inst, err := svc.RegisterInstructor(context.Background(), auth.RegisterInstructorInput{
		FullName: "Alice Instructor",
		Email:    "alice@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Email != "alice@example.com" {
		t.Fatalf("unexpected email: %s", inst.Email)
	}

	var nonce string
	if err := db.QueryRow(`SELECT nonce FROM verification_tokens WHERE instructor_id = ?`, inst.ID).Scan(&nonce); err != nil {
		t.Fatal(err)
	}
	if err := svc.VerifyInstructor(context.Background(), nonce); err != nil {
		t.Fatal(err)
	}
	session, err := svc.Login(context.Background(), "alice@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if session.CSRFToken == "" {
		t.Fatal("expected csrf token")
	}
}
