package database

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"pixelbonus/src/pixelbonus/config"
)

func Open(cfg config.Config, schemaPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", cfg.DatabasePath+"?_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	schema, err := ioutil.ReadFile(schemaPath)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("read schema: %w", err)
	}

	if _, err := db.Exec(string(schema)); err != nil && !strings.Contains(err.Error(), "unsupported exec query") {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return db, nil
}
