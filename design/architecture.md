# PixelBonus Architecture

## Selected architecture

Layered architecture is mandatory because the system requires persistence through SQLite. The HTTP interface is implemented as the presentation layer on top of application services and a storage layer.

## Layers

1. Presentation layer
   - HTTP server, routing, HTML rendering, CSV responses, PDF download responses.
   - Files under `src/pixelbonus/web`, `src/templates`, and `src/assets`.
2. Application layer
   - Deterministic business logic for authentication, courses, coupons, grading, localisation, and asynchronous jobs.
   - Files under `src/pixelbonus/{auth,course,coupon,grade,job,i18n}`.
3. Infrastructure layer
   - SQLite access, PDF rendering, email sending, TLS/server setup, configuration loading.
   - Files under `src/pixelbonus/{database,pdf,email,config}` and `src/main.go`.

## Runtime model

- A single Go process hosts the HTTPS server and one background worker goroutine.
- The process initializes the SQLite schema on startup from `design/schema.sql`.
- Coupon generation is asynchronous:
  1. HTTP handler inserts a queued coupon generation job.
  2. Background worker claims the job in SQLite.
  3. Worker creates token rows, generates the PDF, records the file metadata, and sends the download email.
  4. Worker marks the job completed or failed.
- No external reverse proxy is required.

## Data flow

1. Instructor-facing requests use secure session cookies after login.
2. CSRF-protected POST requests include a synchronizer token stored in the session and echoed in forms.
3. Student coupon redemption writes a single coupon redemption atomically.
4. Grade viewing reads coupon redemption aggregates and applies the configured grading method.

## Concurrency

- SQLite writes are serialized through transactions.
- Coupon redemption uses a conditional update inside a transaction so one coupon can only be redeemed once.
- Background job claiming uses a transaction and a status transition from `queued` to `running`.

## Route map

- `GET /` home page
- `GET /lang/{code}` change interface language
- `GET /instructor/login`
- `POST /instructor/login`
- `GET /instructor/register`
- `POST /instructor/register`
- `GET /instructor/verify`
- `GET /instructor/password/forgot`
- `POST /instructor/password/forgot`
- `GET /instructor/password/reset`
- `POST /instructor/password/reset`
- `POST /instructor/logout`
- `GET /instructor/profile`
- `POST /instructor/profile/password`
- `GET /instructor/courses`
- `GET /instructor/courses/new`
- `POST /instructor/courses`
- `GET /instructor/courses/{courseID}/settings`
- `POST /instructor/courses/{courseID}/settings`
- `GET /instructor/courses/{courseID}/coupons/new`
- `POST /instructor/courses/{courseID}/coupons`
- `GET /instructor/courses/{courseID}/grades`
- `GET /instructor/courses/{courseID}/grades.csv`
- `GET /courses/{courseID}/redemptions`
- `GET /redeem`
- `POST /redeem`
- `GET /r`
- `GET /downloads/{nonce}.pdf`
- `GET /privacy`

## Security model

- HTTPS only, using certificate files from `/etc/letsencrypt/live/pixelbonus.com/fullchain.pem` and `/etc/letsencrypt/live/pixelbonus.com/privkey.pem`.
- Password hashing uses bcrypt.
- Sessions use random 256-bit identifiers stored server-side in SQLite and sent as `Secure`, `HttpOnly`, `SameSite=Lax` cookies.
- CSRF uses a per-session random token embedded in all state-changing forms.
- Email verification and password reset use random 128-bit URL-safe nonces with explicit expiry.
- Student redemption endpoints validate normalized coupon codes and student identifiers before database access.

## Deployment artifacts

- `src/main.go` compiles the single binary.
- `src/pixelbonus.service` is the `systemd` unit.
- `README.md` contains install, run, test, and maintenance instructions including the cron example for removing old PDFs.
