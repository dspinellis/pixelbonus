The system follows a modular monolith architecture implemented as a single Go
binary with clear internal module boundaries around authentication, course
management, coupon generation/redemption, grading/reporting, and shared web
infrastructure.

Modules

1. `src/main.go`
   - Composition root and HTTP server entry point.
   - Loads configuration, initializes schema, wires dependencies, and registers
     routes.

2. `src/platform/config.go`
   - Runtime configuration from environment and filesystem.
   - TLS, database path, mail command path, base URL, and filesystem paths.

3. `src/platform/db.go`
   - SQLite connection, schema initialization, and transaction helpers.

4. `src/platform/httpctx.go`
   - Shared HTTP concerns: routing helpers, template rendering, sessions, CSRF,
     localization selection, and common page layout data.

5. `src/instructors/auth.go`
   - Instructor registration, email confirmation, login, logout, password reset,
     and profile password change.

6. `src/courses/courses.go`
   - Instructor-only course creation, listing, and settings updates.

7. `src/coupons/coupons.go`
   - Coupon token generation, secure download-link generation, asynchronous PDF
     rendering orchestration, and instructor notification email sending.

8. `src/redemption/redemption.go`
   - Student-facing redemption flows by QR link and manual code entry.

9. `src/reporting/reporting.go`
   - Grade calculation, public redemption pages, and CSV export.

10. `src/views/i18n.go` and templates under `src/views/templates/`
    - Localized strings and HTML templates shared by all pages.

Persistence boundaries

- Authentication persists instructors, sessions, confirmation tokens, and
  password-reset tokens.
- Courses persist courses and grading settings.
- Coupons persist token batches, individual tokens, redemptions, and download
  links.
- Reporting reads only through public APIs over the persisted data.

Execution model

- HTTP requests are handled synchronously except PDF generation and emailing for
  token batches, which run in a background worker started by the main process.
- SQLite is the system of record. Generated PDFs are stored on local disk with
  metadata in the database.

Why this architecture

- Satisfies the single-binary requirement.
- Keeps modules testable with clear APIs.
- Avoids cross-module leakage by routing every dependency through explicit
  exported functions/interfaces.
