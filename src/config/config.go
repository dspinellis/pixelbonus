package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type FileReader func(string) (io.ReadCloser, error)

type Config struct {
	Port              int
	LogDir            string
	TemplateDir       string
	BaseURL           string
	DBDSN             string
	PrivacyPolicyURLs map[string]string
}

func Load(env map[string]string, fileReader FileReader) (Config, error) {
	cfg := Config{
		Port:              443,
		LogDir:            ".",
		TemplateDir:       "src/templates",
		BaseURL:           "https://pixelbonus.com",
		DBDSN:             "pixelbonus@tcp(127.0.0.1:3306)/pixelbonus",
		PrivacyPolicyURLs: map[string]string{"en": "/privacy", "el": "/privacy"},
	}
	if fileReader == nil {
		fileReader = func(path string) (io.ReadCloser, error) { return os.Open(path) }
	}
	if path := strings.TrimSpace(env["PIXELBONUS_CONFIG"]); path != "" {
		r, err := fileReader(path)
		if err != nil {
			return Config{}, err
		}
		defer r.Close()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			idx := strings.Index(line, "=")
			if idx <= 0 {
				return Config{}, fmt.Errorf("invalid config line: %s", line)
			}
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			switch key {
			case "port":
				port, err := strconv.Atoi(value)
				if err != nil || port <= 0 || port > 65535 {
					return Config{}, fmt.Errorf("invalid port: %s", value)
				}
				cfg.Port = port
			case "log_dir":
				cfg.LogDir = value
			case "template_dir":
				cfg.TemplateDir = value
			case "base_url":
				cfg.BaseURL = strings.TrimRight(value, "/")
			case "db_dsn":
				cfg.DBDSN = value
			case "privacy_policy_urls.en", "privacy_policy_urls.el":
				if value == "" {
					return Config{}, fmt.Errorf("empty policy URL: %s", key)
				}
				cfg.PrivacyPolicyURLs[strings.TrimPrefix(key, "privacy_policy_urls.")] = value
			default:
				return Config{}, fmt.Errorf("unknown config key: %s", key)
			}
		}
		if err := scanner.Err(); err != nil {
			return Config{}, err
		}
	}
	cfg.LogDir = filepath.Clean(cfg.LogDir)
	cfg.TemplateDir = filepath.Clean(cfg.TemplateDir)
	if value := strings.TrimSpace(env["PIXELBONUS_DB_DSN"]); value != "" {
		cfg.DBDSN = value
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://pixelbonus.com"
	}
	return cfg, nil
}
