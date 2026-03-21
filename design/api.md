# Public Module APIs

All cross-module dependencies must use only the APIs in this file.

## config

- Name: `Load`
- Module path: `src/pixelbonus/config/config.go`
- Full typed signature: `func Load() (Config, error)`
- Behavior: Loads runtime configuration from environment variables with deterministic defaults.
- Error cases: Invalid integer durations, missing required filesystem paths when explicitly overridden.
- Side effects: Reads environment variables.
- Idempotency: Idempotent for unchanged environment.

## database

- Name: `Open`
- Module path: `src/pixelbonus/database/database.go`
- Full typed signature: `func Open(cfg config.Config, schemaPath string) (*sql.DB, error)`
- Behavior: Opens SQLite, configures connection settings, and initializes schema from the SQL file.
- Error cases: SQLite open failure, schema read failure, schema execution failure.
- Side effects: Creates database file and schema objects if absent.
- Idempotency: Idempotent with respect to schema creation.

## i18n

- Name: `NewCatalog`
- Module path: `src/pixelbonus/i18n/i18n.go`
- Full typed signature: `func NewCatalog() Catalog`
- Behavior: Returns the in-memory translation catalog for English and Greek UI strings.
- Error cases: None.
- Side effects: None.
- Idempotency: Idempotent.

- Name: `ResolveLanguage`
- Module path: `src/pixelbonus/i18n/i18n.go`
- Full typed signature: `func ResolveLanguage(cookieValue string, acceptLanguage string) string`
- Behavior: Returns `en` or `el` using the global fallback order.
- Error cases: None.
- Side effects: None.
- Idempotency: Idempotent.

- Name: `T`
- Module path: `src/pixelbonus/i18n/i18n.go`
- Full typed signature: `func (c Catalog) T(lang string, key string) string`
- Behavior: Returns the translated string for the requested language, falling back to English for missing keys.
- Error cases: None.
- Side effects: None.
- Idempotency: Idempotent.

## auth

- Name: `NewService`
- Module path: `src/pixelbonus/auth/service.go`
- Full typed signature: `func NewService(db *sql.DB, sender email.Sender, baseURL string, clock func() time.Time, random io.Reader) *Service`
- Behavior: Creates the authentication service.
- Error cases: None.
- Side effects: None.
- Idempotency: Not applicable.

- Name: `RegisterInstructor`
- Module path: `src/pixelbonus/auth/service.go`
- Full typed signature: `func (s *Service) RegisterInstructor(ctx context.Context, input RegisterInstructorInput) (Instructor, error)`
- Behavior: Creates an unverified instructor, stores a verification token expiring in 24 hours, and sends the verification email.
- Error cases: Duplicate email, invalid input, token generation failure, email send failure, database failure.
- Side effects: Inserts instructor and verification token records; sends email.
- Idempotency: Not idempotent.

- Name: `VerifyInstructor`
- Module path: `src/pixelbonus/auth/service.go`
- Full typed signature: `func (s *Service) VerifyInstructor(ctx context.Context, nonce string) error`
- Behavior: Marks the instructor associated with a valid unexpired verification token as verified and consumes the token.
- Error cases: Unknown token, expired token, database failure.
- Side effects: Updates instructor verification state; deletes or marks token consumed.
- Idempotency: Idempotent after successful verification.

- Name: `Login`
- Module path: `src/pixelbonus/auth/service.go`
- Full typed signature: `func (s *Service) Login(ctx context.Context, email string, password string) (Session, error)`
- Behavior: Authenticates a verified instructor and creates a server-side session with CSRF token.
- Error cases: Invalid credentials, unverified account, database failure, token generation failure.
- Side effects: Inserts session row.
- Idempotency: Not idempotent.

- Name: `Logout`
- Module path: `src/pixelbonus/auth/service.go`
- Full typed signature: `func (s *Service) Logout(ctx context.Context, sessionID string) error`
- Behavior: Invalidates the session if it exists.
- Error cases: Database failure.
- Side effects: Deletes session row.
- Idempotency: Idempotent.

- Name: `RequestPasswordReset`
- Module path: `src/pixelbonus/auth/service.go`
- Full typed signature: `func (s *Service) RequestPasswordReset(ctx context.Context, email string) error`
- Behavior: Creates a one-hour password reset token for an existing verified instructor and sends the reset email; for unknown emails it returns success without sending mail.
- Error cases: Database failure, token generation failure, email send failure.
- Side effects: May insert reset token and send email.
- Idempotency: Not idempotent.

- Name: `ResetPassword`
- Module path: `src/pixelbonus/auth/service.go`
- Full typed signature: `func (s *Service) ResetPassword(ctx context.Context, nonce string, newPassword string) error`
- Behavior: Updates the instructor password for a valid unexpired reset token and consumes the token.
- Error cases: Invalid password, unknown token, expired token, database failure.
- Side effects: Updates password hash; consumes reset token.
- Idempotency: Idempotent after successful reset.

- Name: `ChangePassword`
- Module path: `src/pixelbonus/auth/service.go`
- Full typed signature: `func (s *Service) ChangePassword(ctx context.Context, instructorID int64, newPassword string) error`
- Behavior: Updates the instructor password from the profile page.
- Error cases: Invalid password, unknown instructor, database failure.
- Side effects: Updates password hash.
- Idempotency: Idempotent when the same password hash already matches.

- Name: `InstructorBySession`
- Module path: `src/pixelbonus/auth/service.go`
- Full typed signature: `func (s *Service) InstructorBySession(ctx context.Context, sessionID string) (Instructor, Session, error)`
- Behavior: Returns the verified instructor and session data for a valid session.
- Error cases: Missing session, expired session if expiry support is later introduced, database failure.
- Side effects: None.
- Idempotency: Idempotent.

## course

- Name: `NewService`
- Module path: `src/pixelbonus/course/service.go`
- Full typed signature: `func NewService(db *sql.DB, clock func() time.Time) *Service`
- Behavior: Creates the course service.
- Error cases: None.
- Side effects: None.
- Idempotency: Not applicable.

- Name: `ListCourses`
- Module path: `src/pixelbonus/course/service.go`
- Full typed signature: `func (s *Service) ListCourses(ctx context.Context, instructorID int64) ([]Course, error)`
- Behavior: Returns the instructor's courses ordered by creation time descending.
- Error cases: Database failure.
- Side effects: None.
- Idempotency: Idempotent.

- Name: `CreateCourse`
- Module path: `src/pixelbonus/course/service.go`
- Full typed signature: `func (s *Service) CreateCourse(ctx context.Context, instructorID int64, input CreateCourseInput) (Course, error)`
- Behavior: Creates a course owned by the instructor.
- Error cases: Invalid input, database failure.
- Side effects: Inserts course row.
- Idempotency: Not idempotent.

- Name: `GetCourse`
- Module path: `src/pixelbonus/course/service.go`
- Full typed signature: `func (s *Service) GetCourse(ctx context.Context, instructorID int64, courseID int64) (Course, error)`
- Behavior: Returns one owned course.
- Error cases: Not found, database failure.
- Side effects: None.
- Idempotency: Idempotent.

- Name: `UpdateCourse`
- Module path: `src/pixelbonus/course/service.go`
- Full typed signature: `func (s *Service) UpdateCourse(ctx context.Context, instructorID int64, courseID int64, input UpdateCourseInput) (Course, error)`
- Behavior: Updates course name and grading method.
- Error cases: Invalid input, not found, database failure.
- Side effects: Updates course row.
- Idempotency: Idempotent for identical input.

- Name: `PublicRedemptions`
- Module path: `src/pixelbonus/course/service.go`
- Full typed signature: `func (s *Service) PublicRedemptions(ctx context.Context, courseID int64) (Course, []StudentRedemption, error)`
- Behavior: Returns the course and redemption counts ordered by student identifier.
- Error cases: Not found, database failure.
- Side effects: None.
- Idempotency: Idempotent.

## coupon

- Name: `NewService`
- Module path: `src/pixelbonus/coupon/service.go`
- Full typed signature: `func NewService(db *sql.DB, clock func() time.Time, random io.Reader) *Service`
- Behavior: Creates the coupon service.
- Error cases: None.
- Side effects: None.
- Idempotency: Not applicable.

- Name: `QueueGeneration`
- Module path: `src/pixelbonus/coupon/service.go`
- Full typed signature: `func (s *Service) QueueGeneration(ctx context.Context, instructorID int64, courseID int64, quantity int) (GenerationJob, error)`
- Behavior: Validates the request, ensures the course belongs to the instructor, and inserts a queued job for generating coupon tokens and a PDF.
- Error cases: Invalid quantity, not found, database failure.
- Side effects: Inserts job row.
- Idempotency: Not idempotent.

- Name: `RedeemByCode`
- Module path: `src/pixelbonus/coupon/service.go`
- Full typed signature: `func (s *Service) RedeemByCode(ctx context.Context, code string, studentID string) (RedeemResult, error)`
- Behavior: Normalizes the coupon code, validates the student identifier, atomically redeems an unused coupon, and returns course and coupon details plus the resulting state.
- Error cases: Invalid coupon format, invalid student identifier, unknown coupon, database failure.
- Side effects: Updates one coupon row.
- Idempotency: Idempotent for already redeemed coupons in that the second and subsequent calls return `already redeemed` without changing state.

- Name: `LookupCoupon`
- Module path: `src/pixelbonus/coupon/service.go`
- Full typed signature: `func (s *Service) LookupCoupon(ctx context.Context, code string) (CouponLookup, error)`
- Behavior: Returns the course and normalized code for a valid existing coupon without redeeming it.
- Error cases: Invalid coupon format, unknown coupon, database failure.
- Side effects: None.
- Idempotency: Idempotent.

- Name: `ClaimNextJob`
- Module path: `src/pixelbonus/coupon/service.go`
- Full typed signature: `func (s *Service) ClaimNextJob(ctx context.Context) (GenerationJob, error)`
- Behavior: Atomically changes the oldest queued coupon generation job to running and returns it.
- Error cases: No queued job, database failure.
- Side effects: Updates one job row.
- Idempotency: Not idempotent.

- Name: `CompleteJob`
- Module path: `src/pixelbonus/coupon/service.go`
- Full typed signature: `func (s *Service) CompleteJob(ctx context.Context, jobID int64, tokens []CouponToken, file StoredFile) error`
- Behavior: Persists the generated coupon set, coupon rows, and downloadable file metadata, then marks the job completed.
- Error cases: Database failure.
- Side effects: Inserts coupon set, coupons, file metadata; updates job row.
- Idempotency: Not idempotent.

- Name: `FailJob`
- Module path: `src/pixelbonus/coupon/service.go`
- Full typed signature: `func (s *Service) FailJob(ctx context.Context, jobID int64, message string) error`
- Behavior: Marks a running job failed with a diagnostic message.
- Error cases: Database failure.
- Side effects: Updates job row.
- Idempotency: Idempotent for repeated same message.

- Name: `FileByNonce`
- Module path: `src/pixelbonus/coupon/service.go`
- Full typed signature: `func (s *Service) FileByNonce(ctx context.Context, nonce string) (StoredFile, error)`
- Behavior: Returns the downloadable PDF metadata for a valid file nonce.
- Error cases: Not found, database failure.
- Side effects: None.
- Idempotency: Idempotent.

## grade

- Name: `NewService`
- Module path: `src/pixelbonus/grade/service.go`
- Full typed signature: `func NewService(db *sql.DB) *Service`
- Behavior: Creates the grade service.
- Error cases: None.
- Side effects: None.
- Idempotency: Not applicable.

- Name: `CourseGrades`
- Module path: `src/pixelbonus/grade/service.go`
- Full typed signature: `func (s *Service) CourseGrades(ctx context.Context, instructorID int64, courseID int64) (course.Course, []GradeRow, error)`
- Behavior: Loads redemption counts for one owned course and computes grade and rank for each student.
- Error cases: Not found, database failure.
- Side effects: None.
- Idempotency: Idempotent.

## pdf

- Name: `RenderCoupons`
- Module path: `src/pixelbonus/pdf/render.go`
- Full typed signature: `func RenderCoupons(outputPath string, courseName string, baseURL string, tokens []coupon.CouponToken) error`
- Behavior: Creates an A4 PDF with 20 coupons per page, cut marks, QR codes, grouped token text, and `pixelbonus.com` label.
- Error cases: File creation failure, QR generation failure, PDF write failure.
- Side effects: Creates directories and writes the PDF file.
- Idempotency: Idempotent for identical inputs and path replacement.

## email

- Name: `NewSender`
- Module path: `src/pixelbonus/email/sender.go`
- Full typed signature: `func NewSender(commandPath string, from string) Sender`
- Behavior: Creates the mail sender implementation that uses the Unix `mail` command.
- Error cases: None.
- Side effects: None.
- Idempotency: Not applicable.

- Name: `Send`
- Module path: `src/pixelbonus/email/sender.go`
- Full typed signature: `func (s Sender) Send(to string, subject string, body string) error`
- Behavior: Invokes the Unix mail command with the configured sender.
- Error cases: Command start failure, command exit failure, invalid recipient.
- Side effects: Executes external process.
- Idempotency: Not idempotent.

## job

- Name: `NewWorker`
- Module path: `src/pixelbonus/job/worker.go`
- Full typed signature: `func NewWorker(coupons *coupon.Service, courses *course.Service, mailer email.Sender, cfg config.Config, clock func() time.Time, random io.Reader) *Worker`
- Behavior: Creates the background worker.
- Error cases: None.
- Side effects: None.
- Idempotency: Not applicable.

- Name: `Run`
- Module path: `src/pixelbonus/job/worker.go`
- Full typed signature: `func (w *Worker) Run(ctx context.Context)`
- Behavior: Polls queued jobs until context cancellation, renders PDFs, stores job outputs, and sends notification emails.
- Error cases: Internal failures are persisted through `FailJob`.
- Side effects: Database writes, file writes, email sends.
- Idempotency: Not idempotent across distinct queued jobs.

## web

- Name: `NewServer`
- Module path: `src/pixelbonus/web/handlers.go`
- Full typed signature: `func NewServer(cfg config.Config, catalog i18n.Catalog, authSvc *auth.Service, courseSvc *course.Service, couponSvc *coupon.Service, gradeSvc *grade.Service) http.Handler`
- Behavior: Creates the full HTTP handler tree with all routes and middleware.
- Error cases: None during construction.
- Side effects: None.
- Idempotency: Not applicable.
