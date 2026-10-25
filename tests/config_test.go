package tests

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"pixelbonus/src/config"
)

func TestPrivacyPolicyConfig(t *testing.T) {
	for _, tc := range []struct {
		name, content, en, el string
		wantError             bool
	}{
		{"defaults", "", "/privacy", "/privacy", false},
		{"localized", "privacy_policy_urls.en=https://example.com/en?x=1&y=2\nprivacy_policy_urls.el=https://example.com/el\n", "https://example.com/en?x=1&y=2", "https://example.com/el", false},
		{"partial", "privacy_policy_urls.el=/greek-policy\n", "/privacy", "/greek-policy", false},
		{"empty", "privacy_policy_urls.en=\n", "", "", true},
		{"unsupported language", "privacy_policy_urls.fr=/policy\n", "", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "pixelbonus.conf")
			if err := ioutil.WriteFile(path, []byte(tc.content), 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err := config.Load(map[string]string{"PIXELBONUS_CONFIG": path}, nil)
			if tc.wantError {
				if err == nil {
					t.Fatal("expected invalid policy configuration to fail")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cfg.PrivacyPolicyURLs["en"] != tc.en || cfg.PrivacyPolicyURLs["el"] != tc.el {
				t.Fatalf("unexpected policy URLs: %v", cfg.PrivacyPolicyURLs)
			}
		})
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pixelbonus.conf")
	if err := ioutil.WriteFile(path, []byte("port=8443\nlog_dir=logs\ntemplate_dir=templates\nbase_url=https://example.com\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(map[string]string{"PIXELBONUS_CONFIG": path}, func(path string) (io.ReadCloser, error) { return os.Open(path) })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 8443 || cfg.BaseURL != "https://example.com" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestMySQLConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pixelbonus.conf")
	dsn := "user:p/a=s@s@tcp(localhost:3306)/legacy?tls=true"
	if err := ioutil.WriteFile(path, []byte("db_dsn="+dsn+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(map[string]string{"PIXELBONUS_CONFIG": path}, nil)
	if err != nil || cfg.DBDSN != dsn {
		t.Fatalf("DSN was changed: %q %v", cfg.DBDSN, err)
	}
	override := "other:secret@unix(/tmp/mysql.sock)/newdb"
	cfg, err = config.Load(map[string]string{"PIXELBONUS_CONFIG": path, "PIXELBONUS_DB_DSN": override}, nil)
	if err != nil || cfg.DBDSN != override {
		t.Fatalf("DSN override: %q %v", cfg.DBDSN, err)
	}
}
