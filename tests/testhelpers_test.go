package tests

import (
	"database/sql"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pixelbonus/src/pixelbonus/config"
	"pixelbonus/src/pixelbonus/database"
)

func testConfig(t *testing.T) config.Config {
	t.Helper()
	root := ".."
	tempDir := t.TempDir()
	return config.Config{
		Address:           ":0",
		DatabasePath:      filepath.Join(tempDir, "test.db"),
		SchemaPath:        filepath.Join(root, "design/schema.sql"),
		TemplatesPath:     filepath.Join(root, "src/templates"),
		AssetsPath:        filepath.Join(root, "src/assets"),
		PrivacyPolicyPath: filepath.Join(root, "privacy_policy.md"),
		PDFDir:            filepath.Join(tempDir, "pdf"),
		BaseURL:           "https://pixelbonus.test",
		TLSCertPath:       filepath.Join(tempDir, "cert.pem"),
		TLSKeyPath:        filepath.Join(tempDir, "key.pem"),
		MailCommandPath:   "/bin/true",
		MailFrom:          "no-reply@pixelbonus.com",
		SessionCookieName: "pixelbonus_session",
		LanguageCookie:    "pixelbonus_lang",
	}
}

func testDB(t *testing.T, cfg config.Config) *sql.DB {
	t.Helper()
	db, err := database.Open(cfg, cfg.SchemaPath)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func fixedTime() time.Time {
	return time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
}

func randomReader(char byte) *strings.Reader {
	return strings.NewReader(strings.Repeat(string(char), 8192))
}

func writeMailScript(t *testing.T, output string) string {
	t.Helper()
	script := filepath.Join(t.TempDir(), "mail.sh")
	content := "#!/bin/sh\ncat > \"" + output + "\"\n"
	if err := ioutil.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
	return script
}
