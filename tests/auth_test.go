package tests

import (
	"strings"
	"testing"
	"time"

	"pixelbonus/src/auth"
)

func TestRegisterConfirmLoginAndReset(t *testing.T) {
	db := newTestDB(t)
	mailer := &fakeMailer{}
	svc := &auth.Service{DB: db, Mailer: mailer, BaseURL: "https://pixelbonus.com", Now: func() time.Time { return time.Unix(1000, 0) }}
	reg, err := svc.RegisterInstructor(testContext(), auth.RegisterInput{FullName: "Teacher", Email: "teacher@example.com", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mailer.body, reg.Token) {
		t.Fatalf("verification email missing token: %s", mailer.body)
	}
	if _, err := svc.ConfirmInstructorEmail(testContext(), reg.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(testContext(), "teacher@example.com", "password123"); err != nil {
		t.Fatal(err)
	}
	reset, err := svc.CreatePasswordReset(testContext(), "teacher@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ResetPassword(testContext(), reset.Token, "newpassword123"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(testContext(), "teacher@example.com", "newpassword123"); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyPasswordAndAccountFlags(t *testing.T) {
	db := newTestDB(t)
	id := upsertTestInstructor(t, db)
	// Fixed Symfony SHA-512/5000 fixture for legacy-password{legacy-salt}.
	hash := "JoRVdmbXWLoPVEjh1fyD/hsPHwff/wBSy9fpMnywU4UVux20Vx2tpKJAvy6ZN9vhMMSwlJ+5esq34K8NKABxxA=="
	if _, err := db.Exec("UPDATE Users SET password = ?, salt = ? WHERE id = ?", hash, "legacy-salt", id); err != nil {
		t.Fatal(err)
	}
	svc := &auth.Service{DB: db}
	if _, err := svc.Login(testContext(), "teacher@example.com", "legacy-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(testContext(), "teacher@example.com", "wrong-password"); err != auth.ErrInvalidCredentials {
		t.Fatalf("wrong password accepted: %v", err)
	}
	for _, flag := range []string{"locked", "expired", "credentials_expired"} {
		if _, err := db.Exec("UPDATE Users SET "+flag+" = 1 WHERE id = ?", id); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Login(testContext(), "teacher@example.com", "legacy-password"); err != auth.ErrInvalidCredentials {
			t.Fatalf("flag %s ignored: %v", flag, err)
		}
		if _, err := db.Exec("UPDATE Users SET "+flag+" = 0 WHERE id = ?", id); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.ChangePassword(testContext(), id, "legacy-password", "newpassword123"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(testContext(), "teacher@example.com", "newpassword123"); err != nil {
		t.Fatal(err)
	}
	var stored, salt string
	if err := db.QueryRow("SELECT password,salt FROM Users WHERE id = ?", id).Scan(&stored, &salt); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored, "$2") || salt != "" {
		t.Fatal("password change did not use bcrypt")
	}
	// PHP's bcrypt prefix is accepted without rewriting the stored credential.
	if _, err := db.Exec("UPDATE Users SET password = ? WHERE id = ?", "$2y$"+stored[4:], id); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(testContext(), "teacher@example.com", "newpassword123"); err != nil {
		t.Fatal(err)
	}
}
