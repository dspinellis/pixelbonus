package tests

import "testing"

func TestDatabaseOpenInitializesSchema(t *testing.T) {
	cfg := testConfig(t)
	db := testDB(t, cfg)
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO instructors (full_name, email, password_hash, created_at) VALUES ('A', 'a@example.com', 'x', ?)`, fixedTime()); err != nil {
		t.Fatal(err)
	}
}
