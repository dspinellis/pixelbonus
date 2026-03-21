# API Specification

This document is the sole cross-module reference for package usage.

## `src/config/config.go`

Import path: `pixelbonus/src/config`

- `type Config struct`
  - Holds server address, TLS certificate paths, database path, public base
    URL, cookie settings, mail command path, and filesystem paths for template,
    static, and generated PDF storage.
- `func Load() (Config, error)`
  - Reads runtime configuration from environment variables and reasonable
    defaults for Debian installation.

## `src/store/store.go`

Import path: `pixelbonus/src/store`

- `type Store struct`
  - Wraps the SQLite connection and exposes persistence methods.
- `func Open(path string, schemaPath string) (*Store, error)`
  - Opens the database and applies `schema.sql` initialization.
- `func (s *Store) Close() error`
- `func (s *Store) WithTx(fn func(TxStore) error) error`
  - Runs a callback inside a transaction.
- `type TxStore interface`
  - Transaction-scoped subset of write operations required by service modules.
- `func (s *Store) CreateInstructor(...) (InstructorRecord, error)`
- `func (s *Store) ConfirmInstructor(...) error`
- `func (s *Store) GetInstructorByEmail(email string) (InstructorRecord, error)`
- `func (s *Store) UpdateInstructorPassword(...) error`
- `func (s *Store) CreatePasswordReset(...) (ResetRecord, error)`
- `func (s *Store) ConsumePasswordReset(...) (InstructorRecord, error)`
- `func (s *Store) CreateCourse(...) (CourseRecord, error)`
- `func (s *Store) UpdateCourse(...) error`
- `func (s *Store) ListCoursesByInstructor(instructorID int64) ([]CourseRecord, error)`
- `func (s *Store) GetCourseByID(courseID int64) (CourseRecord, error)`
- `func (s *Store) CreateTokenBatch(...) (TokenBatchRecord, error)`
- `func (s *Store) InsertTokens(batchID int64, tokens []TokenRecord) error`
- `func (s *Store) MarkBatchReady(...) error`
- `func (s *Store) GetTokenByCode(normalizedCode string) (TokenRecord, error)`
- `func (s *Store) RedeemToken(...) (RedemptionRecord, error)`
- `func (s *Store) ListCourseRedemptions(courseID int64) ([]StudentRedemptionRecord, error)`
- `func (s *Store) CreateDownloadLink(...) (DownloadLinkRecord, error)`
- `func (s *Store) GetDownloadLink(secret string) (DownloadLinkRecord, error)`

## `src/auth/auth.go`

Import path: `pixelbonus/src/auth`

- `type Service struct`
- `func NewService(store *store.Store, clock func() time.Time, random io.Reader) *Service`
- `func (s *Service) RegisterInstructor(fullName, email, password, language string) (RegistrationResult, error)`
  - Creates an unconfirmed instructor and a confirmation token.
- `func (s *Service) ConfirmInstructor(token string) error`
- `func (s *Service) Login(email, password string) (Session, error)`
- `func (s *Service) Logout(sessionID string) error`
- `func (s *Service) StartPasswordReset(email string) (PasswordResetResult, error)`
- `func (s *Service) ResetPassword(token, newPassword string) error`
- `func (s *Service) ChangePassword(instructorID int64, currentPassword, newPassword string) error`
- `func (s *Service) ValidateSession(sessionID string) (Session, error)`
- `func (s *Service) NewCSRFToken(sessionID string) (string, error)`
- `func (s *Service) VerifyCSRFToken(sessionID, token string) error`

## `src/course/course.go`

Import path: `pixelbonus/src/course`

- `type Service struct`
- `func NewService(store *store.Store) *Service`
- `func (s *Service) CreateCourse(instructorID int64, name string, gradingMethod string) (Course, error)`
- `func (s *Service) UpdateCourse(courseID, instructorID int64, name string, gradingMethod string) (Course, error)`
- `func (s *Service) ListCourses(instructorID int64) ([]CourseSummary, error)`
- `func (s *Service) GetCourse(courseID, instructorID int64) (Course, error)`
- `func (s *Service) CalculateGrades(courseID, instructorID int64) ([]GradeRow, error)`
- `func (s *Service) ExportGradesCSV(courseID, instructorID int64) ([]byte, error)`
- `func (s *Service) PublicRedemptionBoard(courseID int64) ([]PublicRedemptionRow, error)`

## `src/token/token.go`

Import path: `pixelbonus/src/token`

- `type Service struct`
- `func NewService(store *store.Store, pdfGenerator PDFGenerator, mailer Mailer, clock func() time.Time, random io.Reader, baseURL string) *Service`
- `func (s *Service) CreateBatch(courseID, instructorID int64, count int) (BatchRequestResult, error)`
  - Creates tokens, stores them unused, renders the PDF, creates a permanent
    download link, and emails the instructor.
- `func NormalizeCode(input string) (string, error)`
  - Removes spaces, uppercases hex, and validates length and characters.
- `func FormatCode(normalized string) string`
  - Renders the printable `XXXX XXXX ...` style grouping.
- `func EncodeRedeemURL(baseURL string, normalized string) (string, error)`
- `func (s *Service) ResolveDownload(secret string) (DownloadFile, error)`

## `src/redeem/redeem.go`

Import path: `pixelbonus/src/redeem`

- `type Service struct`
- `func NewService(store *store.Store, clock func() time.Time) *Service`
- `func (s *Service) CouponLanding(code string) (CouponLandingData, error)`
  - Loads the course and printable code for the QR-based landing page.
- `func (s *Service) RedeemByCode(inputCode, studentID string) (RedeemResult, error)`
- `func (s *Service) RedeemByQR(code, studentID string) (RedeemResult, error)`
- `func (s *Service) PublicCourseBoard(courseID int64) ([]PublicRedemptionRow, error)`

## `src/mail/mail.go`

Import path: `pixelbonus/src/mail`

- `type Sender struct`
- `func NewSender(commandPath string, fromAddress string) *Sender`
- `func (s *Sender) Send(to string, subject string, body string) error`

## `src/pdf/pdf.go`

Import path: `pixelbonus/src/pdf`

- `type Generator struct`
- `func NewGenerator(outputDir string) *Generator`
- `func (g *Generator) RenderCoupons(job CouponJob) (RenderedFile, error)`
  - Generates an A4 PDF with 20 coupons per page, cut marks, QR image,
    formatted token code, and `pixelbonus.com` label.

## `src/web/web.go`

Import path: `pixelbonus/src/web`

- `type Server struct`
- `func NewServer(deps Dependencies) (*Server, error)`
- `func (s *Server) Routes() http.Handler`
- `func (s *Server) RenderPrivacy(w http.ResponseWriter, r *http.Request)`
- `func (s *Server) SetLanguage(w http.ResponseWriter, r *http.Request)`
- `func (s *Server) RequireInstructor(next http.Handler) http.Handler`
- `func (s *Server) CurrentSession(r *http.Request) (auth.Session, bool)`

## `src/app/app.go`

Import path: `pixelbonus/src/app`

- `type App struct`
- `func New(cfg config.Config) (*App, error)`
- `func (a *App) Handler() http.Handler`
- `func (a *App) Run(ctx context.Context) error`

## `src/main.go`

- `func main()`
  - Loads config, constructs the app, and starts the HTTPS server.
