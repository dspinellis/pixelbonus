# Module: database

- Source files: `src/database/mysql.go`, `src/database/transaction.go`
- APIs: `OpenMySQL`, `DateTime`, `WithTx`
- Opens a pooled MySQL connection using a DSN; parses legacy DATETIME values as strings.
- Adds missing tables from `design/schema.sql` without changing existing tables or data.
- Explicit initialization can create a missing database; normal startup selects an existing database.
- Formats legacy DATETIME writes as UTC and wraps transactional callbacks.
- Uses `github.com/go-sql-driver/mysql` v1.7.1, compatible with Go 1.19.
- SQLite is confined to tests and mirrors the legacy domain schema in `tests/sqlite-schema.sql`.
