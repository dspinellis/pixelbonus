# Architecture

The system is a monolithic Go 1.15 web application organized into
testable internal modules behind explicit package APIs. The application
will be compiled as a single static binary and will use SQLite for
persistence.

## Architectural style

- Server-side rendered web application using `net/http` and HTML templates.
- Layered design:
  - HTTP delivery layer for routing, request decoding, response rendering,
    session cookies, CSRF checks, and language selection.
  - Application services layer for instructor, course, token, redemption,
    grading, and download workflows.
  - Persistence layer for SQLite access and schema initialization.
  - Infrastructure layer for email delivery, PDF generation, QR generation,
    and TLS/system integration.

## Modules

- `src/app/app.go`
  - Application wiring, route registration, startup, and shared dependencies.
- `src/config/config.go`
  - Runtime configuration, filesystem paths, and environment parsing.
- `src/store/store.go`
  - SQLite schema initialization and all database operations.
- `src/auth/auth.go`
  - Instructor registration, confirmation, login, password reset, password
    change, secure session management, and CSRF primitives.
- `src/course/course.go`
  - Course lifecycle management, grading method handling, ranking, and CSV
    export formatting.
- `src/token/token.go`
  - Token generation, normalization, QR payload encoding, PDF job creation,
    and secure download link management.
- `src/redeem/redeem.go`
  - Student coupon redemption by QR link or manual code entry and public
    redemption reporting.
- `src/web/web.go`
  - HTTP handlers, templates, localization catalogs, form validation
    feedback, and page composition.
- `src/mail/mail.go`
  - Email composition and delivery through the Unix `mail` command.
- `src/pdf/pdf.go`
  - Coupon PDF rendering and QR image composition.

## Data flow

1. `main.go` loads configuration and constructs the application through
   `app.New`.
2. `app` initializes the SQLite schema through `store`, loads translation
   catalogs through `web`, and starts the HTTP server.
3. HTTP handlers in `web` call only the package APIs listed in `specs/api.md`.
4. State-changing workflows use `auth`, `course`, `token`, and `redeem`,
   which persist data through `store`.
5. Token batch creation calls `pdf` to render a coupon document and `mail`
   to notify the instructor with a permanent download URL.

## External dependencies

- Standard library by default.
- A minimal QR generation library is allowed because QR creation is explicitly
  required and the standard library does not provide it.
- SQLite will be accessed without an ORM.

## Source layout

- `src/main.go` is the executable entry point.
- `src/...` contains all application source packages.
- `tests/...` contains unit and integration tests.
- `schema.sql` stores the database schema used by automatic initialization.
- `README.md` documents build, run, deploy, test, and monitoring operations.
- `pixelbonus.service` contains the systemd unit file.
