# Global Specification

## Language and Version

- Language: Go
- Target version: Go 1.19

## Dependency Policy

- Use the Go standard library by default.
- External dependencies are allowed only where the specification makes them necessary:
  - SQLite driver
  - bcrypt password hashing
  - QR code generation
  - PDF generation if standard-library-only PDF creation would be unreasonably complex
- No ORM.
- No JS frameworks.
- No vendoring.
- `go mod tidy` must resolve all modules without `replace` directives.

## Runtime Configuration

An optional configuration file path is read from environment variable `PIXELBONUS_CONFIG`.
Supported configuration keys:
- `port`
- `log_dir`
- `template_dir`
- `base_url`

Defaults when absent:
- `port = 443`
- `log_dir = .`
- `template_dir = src/templates`
- `base_url = https://pixelbonus.com`

## Error Handling Strategy

- Validation errors return deterministic field-level messages.
- Authentication failures return generic messages that do not disclose whether an account exists, except where the spec explicitly requires acknowledgement regardless of account existence.
- Database errors, file errors, and mail errors are logged and surfaced as generic localized error pages.
- Background job failures are persisted on the token set row and logged; the instructor can request a fresh set instead of mutating failed output.

## Logging

- Log destination: `${log_dir}/pixelbonus.log`
- Log format: one line per event with UTC timestamp, severity, event code, and key fields.
- Log sensitive data policy:
  - do not log passwords
  - do not log raw password reset nonces
  - do not log full session secrets
  - token codes may be logged only by token-set identifier, not by raw token value

## Internationalization

- Every user-visible string exists in both `en` and `el` catalogs.
- Templates receive translation keys or translated strings from the `i18n` module only.
- Language resolution order:
  1. explicit form selection
  2. persisted language cookie
  3. `Accept-Language`
  4. English

## Data Validation

- Instructor full name: 1 to 120 Unicode scalar values after trimming.
- Instructor email: normalized to lowercase and validated with `net/mail`.
- Password: minimum 8 characters.
- Student identifier: 1 to 20 characters after trimming.
- Course name: 1 to 120 characters after trimming.
- Token batch size: positive multiple of 20.
- Coupon code input: exactly 40 hex characters after removing spaces, case-insensitive.

## Persistence Conventions

- All timestamps stored in UTC RFC3339-compatible SQLite text values.
- Nonces stored as URL-safe base64 without padding unless hex is explicitly required.
- Coupon tokens are stored as uppercase 40-character hex strings.

## Implementation-Defined Choices Fixed By Design

- Cookie names:
  - language: `pixelbonus_lang`
  - session: `pixelbonus_session`
- Session expiration: 12 hours idle, 7 days absolute.
- Verification token encoding: URL-safe base64 without padding.
- Password reset token encoding: URL-safe base64 without padding.
- Public course nonce encoding: URL-safe base64 without padding.
- Secure download file nonce: basename of the generated PDF file without `.pdf`.
- Ranking policy: standard competition ranking on redemption count descending, with ties sharing the same rank and the next rank skipping ahead.
- Quartile calculation: nearest-rank quartiles on the sorted redemption counts of students with at least one redemption.
- Grade interpolation: piecewise linear interpolation between configured anchor points, rounded to nearest integer with halves rounded up, clamped to `[0,10]`.
- Redemption winner for duplicate submission race: the first transaction to atomically mark the token as redeemed.
