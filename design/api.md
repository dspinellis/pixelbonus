# API Specification

## config

### Name
`Load`

- Module path: `src/config`
- Signature: `func Load(env map[string]string, fileReader FileReader) (Config, error)`
- Behavior: loads defaults, optionally merges configuration file referenced by `PIXELBONUS_CONFIG`, normalizes paths, and returns immutable runtime configuration.
- Error cases: unreadable config file; malformed `key=value` line; invalid port.
- Side effects: reads one optional file.
- Idempotency: idempotent for identical inputs.

## logging

### Name
`New`

- Module path: `src/logging`
- Signature: `func New(logDir string, now func() time.Time) (*Logger, error)`
- Behavior: opens or creates `${logDir}/pixelbonus.log` in append mode and returns a concurrency-safe logger.
- Error cases: directory creation failure; file open failure.
- Side effects: creates directory or file.
- Idempotency: not idempotent because repeated calls create independent logger instances.

### Name
`Log`

- Module path: `src/logging`
- Signature: `func (l *Logger) Log(level string, event string, fields map[string]string)`
- Behavior: appends one formatted log line.
- Error cases: none surfaced; write errors are ignored after best-effort attempt.
- Side effects: writes to the log file.
- Idempotency: not idempotent.

## i18n

### Name
`LoadCatalogs`

- Module path: `src/i18n`
- Signature: `func LoadCatalogs(dir string) (Catalogs, error)`
- Behavior: loads `en.json` and `el.json` from `dir`, verifies both contain the same keys, and returns lookup catalogs.
- Error cases: missing file; invalid JSON; mismatched keys.
- Side effects: reads translation files from disk.
- Idempotency: idempotent for identical file content.

### Name
`ResolveLanguage`

- Module path: `src/i18n`
- Signature: `func ResolveLanguage(explicit string, cookie string, acceptLanguage string) string`
- Behavior: returns `en` or `el` using the precedence defined in `design/global.md`.
- Error cases: none.
- Side effects: none.
- Idempotency: idempotent.

### Name
`T`

- Module path: `src/i18n`
- Signature: `func (c Catalogs) T(lang string, key string, data map[string]string) string`
- Behavior: returns the translated string for `lang` and interpolates `{{key}}` placeholders from `data`.
- Error cases: missing key panics during startup validation and never at runtime.
- Side effects: none.
- Idempotency: idempotent.

## database

### Name
`OpenMySQL`

- Module path: `src/database`
- Signature: `func OpenMySQL(dsn string, schema string, createDatabase bool) (*sql.DB, error)`
- Behavior: opens a pooled MySQL connection, optionally creates the selected database, adds missing tables without altering existing legacy tables, and returns the handle.
- Error cases: driver open failure; schema execution failure.
- Side effects: creates the database file if missing; mutates schema state.
- Idempotency: idempotent with respect to schema creation.

### Name
`WithTx`

- Module path: `src/database`
- Signature: `func WithTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error`
- Behavior: starts a serializable-equivalent immediate transaction, invokes `fn`, commits on nil, and rolls back on error.
- Error cases: begin failure; callback error; commit failure.
- Side effects: mutates database state when callback succeeds.
- Idempotency: depends on callback.

## auth

### Name
`RegisterInstructor`

- Module path: `src/auth`
- Signature: `func (s *Service) RegisterInstructor(ctx context.Context, input RegisterInput) (RegistrationResult, error)`
- Behavior: validates instructor data, enforces unique email, persists an unverified instructor plus a verification token expiring in 24 hours, and returns email payload metadata.
- Error cases: invalid name/email/password; duplicate email; persistence failure.
- Side effects: writes instructor and verification rows.
- Idempotency: not idempotent.

### Name
`ConfirmInstructorEmail`

- Module path: `src/auth`
- Signature: `func (s *Service) ConfirmInstructorEmail(ctx context.Context, token string) (ConfirmResult, error)`
- Behavior: verifies a non-expired token, marks the instructor verified, and invalidates the token.
- Error cases: token missing; token expired; persistence failure.
- Side effects: updates instructor and verification token rows.
- Idempotency: idempotent after successful confirmation by returning already-confirmed status.

### Name
`Login`

- Module path: `src/auth`
- Signature: `func (s *Service) Login(ctx context.Context, email string, password string) (InstructorIdentity, error)`
- Behavior: authenticates a verified instructor by email and password.
- Error cases: invalid credentials; unverified account; persistence failure.
- Side effects: none.
- Idempotency: idempotent.

### Name
`CreatePasswordReset`

- Module path: `src/auth`
- Signature: `func (s *Service) CreatePasswordReset(ctx context.Context, email string) (PasswordResetResult, error)`
- Behavior: if an instructor exists, creates a one-hour reset token and returns email metadata; otherwise returns an acknowledged no-op result.
- Error cases: persistence failure.
- Side effects: may write a reset token row.
- Idempotency: not idempotent for existing accounts.

### Name
`ResetPassword`

- Module path: `src/auth`
- Signature: `func (s *Service) ResetPassword(ctx context.Context, token string, newPassword string) (ResetResult, error)`
- Behavior: validates a reset token and password, updates the stored password hash, and invalidates the token.
- Error cases: invalid password; token missing; token expired; persistence failure.
- Side effects: updates password hash and token rows.
- Idempotency: not idempotent.

### Name
`ChangePassword`

- Module path: `src/auth`
- Signature: `func (s *Service) ChangePassword(ctx context.Context, instructorID int64, currentPassword string, newPassword string) error`
- Behavior: verifies the current password and updates the password hash.
- Error cases: invalid credentials; invalid password; persistence failure.
- Side effects: updates instructor row.
- Idempotency: not idempotent.

## session

### Name
`Create`

- Module path: `src/session`
- Signature: `func (s *Store) Create(ctx context.Context, instructorID int64, now time.Time) (SessionData, error)`
- Behavior: creates a persisted session record and returns the cookie payload.
- Error cases: randomness failure; persistence failure.
- Side effects: writes session row.
- Idempotency: not idempotent.

### Name
`Validate`

- Module path: `src/session`
- Signature: `func (s *Store) Validate(ctx context.Context, cookieValue string, now time.Time) (SessionData, error)`
- Behavior: verifies cookie signature, loads the session, enforces expiry, rotates last-seen timestamp, and returns session data.
- Error cases: malformed cookie; bad signature; missing session; expired session; persistence failure.
- Side effects: updates session last-seen timestamp.
- Idempotency: not idempotent because it updates last-seen.

### Name
`Destroy`

- Module path: `src/session`
- Signature: `func (s *Store) Destroy(ctx context.Context, cookieValue string) error`
- Behavior: removes the persisted session if present.
- Error cases: persistence failure.
- Side effects: deletes session row.
- Idempotency: idempotent.

## csrf

### Name
`Issue`

- Module path: `src/csrf`
- Signature: `func (s *Service) Issue(sessionID string) (string, error)`
- Behavior: creates a token deterministically bound to the session and current secret generation.
- Error cases: randomness failure.
- Side effects: none.
- Idempotency: not idempotent.

### Name
`Verify`

- Module path: `src/csrf`
- Signature: `func (s *Service) Verify(sessionID string, token string) bool`
- Behavior: returns true only for valid, non-expired token format issued for the session.
- Error cases: none.
- Side effects: none.
- Idempotency: idempotent.

## mail

### Name
`Send`

- Module path: `src/mail`
- Signature: `func (s *Service) Send(ctx context.Context, to string, subject string, body string) error`
- Behavior: executes the Unix `mail` command with sender `no-reply@pixelbonus.com`.
- Error cases: invalid address; process execution failure; non-zero exit status.
- Side effects: spawns external process.
- Idempotency: not idempotent.

## courses

### Name
`CreateCourse`

- Module path: `src/courses`
- Signature: `func (s *Service) CreateCourse(ctx context.Context, instructorID int64, input CreateCourseInput) (Course, error)`
- Behavior: validates input, persists the course, assigns a public nonce, and returns the created course.
- Error cases: invalid name; invalid grading method; persistence failure.
- Side effects: writes course row.
- Idempotency: not idempotent.

### Name
`ListCourses`

- Module path: `src/courses`
- Signature: `func (s *Service) ListCourses(ctx context.Context, instructorID int64) ([]Course, error)`
- Behavior: returns the instructor's courses ordered by creation time descending.
- Error cases: persistence failure.
- Side effects: none.
- Idempotency: idempotent.

### Name
`GetCourse`

- Module path: `src/courses`
- Signature: `func (s *Service) GetCourse(ctx context.Context, instructorID int64, courseID int64) (Course, error)`
- Behavior: returns one course owned by the instructor.
- Error cases: not found; persistence failure.
- Side effects: none.
- Idempotency: idempotent.

### Name
`UpdateCourse`

- Module path: `src/courses`
- Signature: `func (s *Service) UpdateCourse(ctx context.Context, instructorID int64, input UpdateCourseInput) (Course, error)`
- Behavior: validates and updates course name and grading method.
- Error cases: invalid input; not found; persistence failure.
- Side effects: updates course row.
- Idempotency: idempotent for identical inputs.

## tokens

### Name
`CreateTokenSet`

- Module path: `src/tokens`
- Signature: `func (s *Service) CreateTokenSet(ctx context.Context, instructorID int64, courseID int64, count int, now time.Time) (TokenSet, error)`
- Behavior: validates ownership and count, creates a token-set row, generates `count` unique 160-bit token codes, persists unused tokens, allocates a file nonce, marks the job pending, and schedules background PDF generation.
- Error cases: invalid count; course not found; randomness failure; persistence failure.
- Side effects: writes token-set and token rows; triggers background job.
- Idempotency: not idempotent.

### Name
`ResumePendingJobs`

- Module path: `src/tokens`
- Signature: `func (s *Service) ResumePendingJobs(ctx context.Context) error`
- Behavior: loads token sets in `pending` status and reschedules background generation.
- Error cases: persistence failure.
- Side effects: schedules background work.
- Idempotency: idempotent with duplicate-suppression by job status.

### Name
`GetTokenByCode`

- Module path: `src/tokens`
- Signature: `func (s *Service) GetTokenByCode(ctx context.Context, normalizedCode string) (CouponToken, error)`
- Behavior: returns token and course data for a normalized 40-hex-character code.
- Error cases: invalid format; not found; persistence failure.
- Side effects: none.
- Idempotency: idempotent.

### Name
`GetDownloadPath`

- Module path: `src/tokens`
- Signature: `func (s *Service) GetDownloadPath(ctx context.Context, fileNonce string) (string, string, error)`
- Behavior: resolves a generated token-set file nonce to the absolute PDF path and original course name for download.
- Error cases: not found; job incomplete; persistence failure.
- Side effects: none.
- Idempotency: idempotent.

## pdf

### Name
`GenerateCoupons`

- Module path: `src/pdf`
- Signature: `func (s *Service) GenerateCoupons(ctx context.Context, outputPath string, redeemBaseURL string, tokens []CouponPrintData) error`
- Behavior: creates an A4 PDF with 20 coupons per page, cut marks, QR codes, formatted coupon code groups, and the `pixelbonus.com` label.
- Error cases: QR generation failure; output directory creation failure; file write failure.
- Side effects: creates directories and writes a PDF file.
- Idempotency: idempotent for identical output path and input data by overwriting.

## redemption

### Name
`Redeem`

- Module path: `src/redemption`
- Signature: `func (s *Service) Redeem(ctx context.Context, input RedeemInput, now time.Time) (RedeemResult, error)`
- Behavior: normalizes the token code, validates the student identifier, atomically marks an unused token redeemed to the provided student identifier, records redemption timestamp, and returns success, already-redeemed, or invalid result with course metadata.
- Error cases: invalid student identifier; invalid code format; persistence failure.
- Side effects: updates token row and inserts redemption row on success.
- Idempotency: idempotent for repeated submission of an already-redeemed token because status remains already redeemed.

### Name
`LookupRedeemPage`

- Module path: `src/redemption`
- Signature: `func (s *Service) LookupRedeemPage(ctx context.Context, encodedToken string) (RedeemPageData, error)`
- Behavior: decodes the URL-safe token from `/r`, resolves the token and course, and returns read-only display data.
- Error cases: invalid encoding; unknown token; persistence failure.
- Side effects: none.
- Idempotency: idempotent.

## grades

### Name
`ComputeCourseGrades`

- Module path: `src/grades`
- Signature: `func (s *Service) ComputeCourseGrades(ctx context.Context, instructorID int64, courseID int64) (CourseGrades, error)`
- Behavior: loads per-student redemption counts for the owned course, computes quartiles, ranks, grades, and returns rows ordered by student identifier.
- Error cases: course not found; persistence failure.
- Side effects: none.
- Idempotency: idempotent.

### Name
`RenderCSV`

- Module path: `src/grades`
- Signature: `func RenderCSV(rows []GradeRow) ([]byte, error)`
- Behavior: renders deterministic CSV with header `student_id,redemptions,grade,rank`.
- Error cases: write buffer failure.
- Side effects: none.
- Idempotency: idempotent.

## public

### Name
`ListPublicRedemptions`

- Module path: `src/public`
- Signature: `func (s *Service) ListPublicRedemptions(ctx context.Context, publicNonce string) (PublicCourseView, error)`
- Behavior: resolves a course by public nonce and returns rows ordered by student identifier containing student identifier and redemption count.
- Error cases: not found; persistence failure.
- Side effects: none.
- Idempotency: idempotent.

## web

### Name
`NewServer`

- Module path: `src/web`
- Signature: `func NewServer(deps Dependencies) (*Server, error)`
- Behavior: parses templates, registers routes, and returns an HTTP server facade.
- Error cases: template parse failure; missing dependencies.
- Side effects: reads template files.
- Idempotency: not idempotent because it allocates a new server.

### Name
`Routes`

- Module path: `src/web`
- Signature: `func (s *Server) Routes() http.Handler`
- Behavior: returns the fully wired HTTP handler with middleware for logging, language resolution, sessions, CSRF, and authentication.
- Error cases: none.
- Side effects: none.
- Idempotency: idempotent.
