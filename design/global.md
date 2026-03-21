# Global Specification

## Language and version

- Language: Go
- Minimum language level: Go 1.15 compatible source

## Dependency policy

- Standard library first.
- Allowed external dependencies because they are necessary:
  - SQLite driver for `database/sql`
  - bcrypt password hashing
  - QR code generation
- No ORM.
- No JavaScript framework.

## Error handling strategy

- Internal APIs return typed Go errors.
- HTTP handlers map errors deterministically:
  - validation errors -> `400 Bad Request` with rendered message
  - authentication failures -> `401 Unauthorized` or login form error
  - authorization failures -> `403 Forbidden`
  - missing resources -> `404 Not Found`
  - state conflicts, such as already redeemed coupons -> `409 Conflict` when returned as HTTP status, while user-facing redemption pages still render the exact required success/error text
  - unexpected failures -> `500 Internal Server Error`
- Background jobs persist failure messages in SQLite and never panic the process.

## Global implementation-defined choices

- Language cookie name: `pixelbonus_lang`
- Session cookie name: `pixelbonus_session`
- CSRF field name: `csrf_token`
- Instructor verification token lifetime: 24 hours
- Password reset token lifetime: 1 hour
- Secure PDF download nonce length: 128 bits, lowercase hex filename
- Coupon token format for user display: uppercase hexadecimal grouped as `xxxx xxxx xxxx xxxx xxxx xxxx xxxx xxxx xxxx xxxx`
- Coupon token storage format: 20 raw bytes plus normalized uppercase hexadecimal text cache
- Grading tie handling:
  - Quartiles are selected from the sorted redemption counts using nearest-rank with `ceil(p*n)` and 1-based indexing.
  - Ranks are competition ranks (`1, 2, 2, 4`) ordered by redemption count descending then student identifier ascending.
- Language fallback order: explicit cookie, `Accept-Language`, English
- Student identifier normalization: trim surrounding whitespace; preserve internal characters; reject empty values and values longer than 20 Unicode code points

## Non-functional requirements

- Single process, single static binary target.
- SQLite schema initialized automatically at startup.
- Templates and translation strings are stored in files under `src`.
- Readable URLs are used for all routes.
- Stand-alone HTTPS server, compatible with Debian stable and `systemd`.

## Persistence requirement

- SQLite database file path is configurable.
- Schema is defined only by `design/schema.sql`.
- All schema updates in this repository are bootstrap-only; no migration framework is introduced.
