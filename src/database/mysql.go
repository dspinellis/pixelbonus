package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// OpenMySQL adds missing application tables without altering existing legacy tables.
// createDatabase is intended for an explicit fresh-install initialization command.
func OpenMySQL(dsn, schema string, createDatabase bool) (*sql.DB, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid MySQL DSN")
	}
	if cfg.DBName == "" {
		return nil, fmt.Errorf("MySQL DSN must select a database")
	}
	// Legacy DATETIME columns are read as strings; application token expiry values
	// deliberately use RFC3339 strings in separate tables.
	cfg.ParseTime = false
	cfg.MultiStatements = false
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 30 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 30 * time.Second
	}
	if createDatabase {
		admin := *cfg
		admin.DBName = ""
		db, err := sql.Open("mysql", admin.FormatDSN())
		if err != nil {
			return nil, err
		}
		_, err = db.Exec("CREATE DATABASE IF NOT EXISTS `" + strings.ReplaceAll(cfg.DBName, "`", "``") + "` CHARACTER SET utf8 COLLATE utf8_unicode_ci")
		db.Close()
		if err != nil {
			return nil, fmt.Errorf("create database: %w", err)
		}
	}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(3 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err == nil {
		// The shipped schema contains only simple CREATE TABLE statements.
		for _, statement := range strings.Split(schema, ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, err = db.ExecContext(ctx, statement); err != nil {
				break
			}
		}
	}
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize MySQL: %w", err)
	}
	return db, nil
}

// DateTime formats values for the legacy MySQL DATETIME columns (UTC).
func DateTime(t time.Time) string { return t.UTC().Format("2006-01-02 15:04:05") }
