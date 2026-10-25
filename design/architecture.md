# Architecture

## Selection

The system uses a layered MVC/REST architecture because the specification requires both persistence and request/response web interactions.

## Layers

1. Presentation layer
   - HTTP routes
   - HTML rendering
   - form decoding and validation
   - language and session selection
2. Application layer
   - instructor registration and login
   - password reset
   - course creation and updates
   - token generation
   - redemption
   - grade calculation
   - public redemption views
3. Infrastructure layer
   - SQLite persistence
   - translation file loading
   - secure cookie sessions
   - CSRF token issuance and verification
   - email delivery via the Unix `mail` command
   - PDF and QR generation
   - TLS server startup
   - file logging

## Module Boundaries

- `config`: loads environment-driven runtime configuration.
- `logging`: owns structured action logging to `pixelbonus.log`.
- `i18n`: owns translation catalogs and language selection.
- `database`: owns schema initialization and SQLite access helpers.
- `auth`: owns instructor accounts, email verification, password reset, and authentication.
- `session`: owns signed secure cookie sessions.
- `csrf`: owns request-scoped CSRF tokens bound to sessions.
- `mail`: owns email composition and invocation of `/usr/bin/mail`.
- `courses`: owns course CRUD and course listing.
- `tokens`: owns token generation, token-set persistence, file-link generation, and coupon generation jobs.
- `grades`: owns quartile-based grading and CSV export data.
- `redemption`: owns student-facing token lookup and redemption.
- `public`: owns public course redemption listings.
- `pdf`: owns A4 coupon PDF generation and QR embedding.
- `web`: owns routes, handlers, template rendering, and integration of all application services.
- `app`: binary wiring, startup, shutdown, and TLS listen loop.

## Dependency Rules

- Presentation depends only on application APIs declared in `design/api.md`.
- Application services depend on repository-facing APIs declared in `design/api.md`.
- No module reads another module's internals directly.
- Templates never contain hardcoded user-visible strings; they only receive translation keys or resolved text from `i18n`.

## Persistence Model

SQLite stores:
- instructors
- email verification tokens
- password reset tokens
- sessions
- courses
- public course nonce mappings
- token sets
- coupon tokens
- redemptions

Schema initialization runs automatically during startup before the HTTP server begins listening.

## Async Work

Coupon generation is asynchronous within the single process.
- HTTP request creates the token set and persists placeholder metadata.
- Background goroutine generates the PDF file.
- Completion path updates the token set with file metadata and sends the email notification.
- On restart, unfinished jobs remain persisted as `pending`; a startup recovery scan resumes them.

## Route Style

Readable routes:
- `/`
- `/lang`
- `/privacy`
- `/redeem`
- `/r`
- `/instructor/login`
- `/instructor/register`
- `/instructor/register/confirm`
- `/instructor/password/forgot`
- `/instructor/password/reset`
- `/instructor/profile`
- `/instructor/logout`
- `/instructor/courses`
- `/instructor/courses/new`
- `/instructor/courses/{courseID}/settings`
- `/instructor/courses/{courseID}/tokens/new`
- `/instructor/courses/{courseID}/grades`
- `/instructor/courses/{courseID}/grades.csv`
- `/public/{publicNonce}`
- `/download/{fileNonce}`

## Security Model

- TLS terminates in-process using the provided Let's Encrypt files.
- Passwords are hashed with bcrypt.
- Sessions use `HttpOnly`, `Secure`, and `SameSite=Lax` cookies.
- CSRF tokens are required on all state-changing HTML form submissions.
- Instructor-only routes require an authenticated instructor session and verified account.
- Student redemption requires no authentication.
- Token redemption is atomic in the database to ensure a token is redeemed at most once.
