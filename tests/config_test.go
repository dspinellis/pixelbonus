package tests

import (
	"os"
	"testing"

	"pixelbonus/src/pixelbonus/config"
)

func TestLoadConfigDefaults(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("PIXELBONUS_SCHEMA", "../design/schema.sql")
	os.Setenv("PIXELBONUS_TEMPLATES", "../src/templates")
	os.Setenv("PIXELBONUS_ASSETS", "../src/assets")
	os.Setenv("PIXELBONUS_PRIVACY", "../privacy_policy.md")
	os.Setenv("PIXELBONUS_PDF_DIR", tempDir)
	defer os.Clearenv()

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PDFDir != tempDir {
		t.Fatalf("unexpected pdf dir: %s", cfg.PDFDir)
	}
}
