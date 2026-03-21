package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Address           string
	DatabasePath      string
	SchemaPath        string
	TemplatesPath     string
	AssetsPath        string
	PrivacyPolicyPath string
	PDFDir            string
	BaseURL           string
	TLSCertPath       string
	TLSKeyPath        string
	MailCommandPath   string
	MailFrom          string
	SessionCookieName string
	LanguageCookie    string
}

func Load() (Config, error) {
	cfg := Config{
		Address:           getenv("PIXELBONUS_ADDR", ":8443"),
		DatabasePath:      getenv("PIXELBONUS_DB", "./pixelbonus.db"),
		SchemaPath:        getenv("PIXELBONUS_SCHEMA", "./design/schema.sql"),
		TemplatesPath:     getenv("PIXELBONUS_TEMPLATES", "./src/templates"),
		AssetsPath:        getenv("PIXELBONUS_ASSETS", "./src/assets"),
		PrivacyPolicyPath: getenv("PIXELBONUS_PRIVACY", "./privacy_policy.md"),
		PDFDir:            getenv("PIXELBONUS_PDF_DIR", "/var/tmp/pixelbonus"),
		BaseURL:           getenv("PIXELBONUS_BASE_URL", "https://pixelbonus.com"),
		TLSCertPath:       getenv("PIXELBONUS_TLS_CERT", "/etc/letsencrypt/live/pixelbonus.com/fullchain.pem"),
		TLSKeyPath:        getenv("PIXELBONUS_TLS_KEY", "/etc/letsencrypt/live/pixelbonus.com/privkey.pem"),
		MailCommandPath:   getenv("PIXELBONUS_MAIL_CMD", "/usr/bin/mail"),
		MailFrom:          "no-reply@pixelbonus.com",
		SessionCookieName: "pixelbonus_session",
		LanguageCookie:    "pixelbonus_lang",
	}

	if cfg.Address == "" {
		return Config{}, fmt.Errorf("address must not be empty")
	}
	for _, p := range []string{cfg.SchemaPath, cfg.TemplatesPath, cfg.AssetsPath, cfg.PrivacyPolicyPath} {
		if _, err := os.Stat(p); err != nil {
			return Config{}, fmt.Errorf("path %s: %w", p, err)
		}
	}
	if err := os.MkdirAll(filepath.Clean(cfg.PDFDir), 0755); err != nil {
		return Config{}, fmt.Errorf("create pdf dir: %w", err)
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
