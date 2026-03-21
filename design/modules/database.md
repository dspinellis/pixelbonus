# Module Specification: database

- Target source file path: `src/pixelbonus/database/database.go`
- Owned APIs:
  - `Open`
- Internal logic:
  - Open SQLite through `database/sql`.
  - Set a busy timeout and foreign keys pragma on every connection.
  - Read schema SQL from the supplied path and execute it.
  - Use `CREATE TABLE IF NOT EXISTS` schema so repeated startup is safe.
- Dependencies ONLY via `design/api.md`:
  - `config.Load`
