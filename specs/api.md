This file defines the APIs exposed by each implementation file. Import paths are
relative to the module root of the generated Go application.

`src/platform/config.go`
APIs: `LoadConfig`
- Import: `pixelbonus/src/platform`
- `LoadConfig() (Config, error)`
  Returns validated runtime configuration.

`src/platform/db.go`
APIs: `OpenDatabase`, `InitializeSchema`, `WithTx`
- Import: `pixelbonus/src/platform`
- `OpenDatabase(cfg Config) (*sql.DB, error)`
- `InitializeSchema(db *sql.DB, schemaPath string) error`
- `WithTx(db *sql.DB, fn func(*sql.Tx) error) error`

`src/platform/httpctx.go`
APIs: `NewServerContext`, `RegisterSharedRoutes`, `RequireInstructor`
- Import: `pixelbonus/src/platform`
- `NewServerContext(cfg Config, db *sql.DB, deps ServerDeps) (*ServerContext, error)`
  Creates the shared HTTP context, template renderer, session manager, CSRF
  helpers, localization support, and background services.
- `RegisterSharedRoutes(mux *http.ServeMux, ctx *ServerContext)`
  Registers home page, language switcher, privacy statement, and static asset
  routes.
- `RequireInstructor(next http.Handler, ctx *ServerContext) http.Handler`
  Enforces authenticated instructor access.

`src/instructors/auth.go`
APIs: `RegisterAuthRoutes`, `AuthService`
- Import: `pixelbonus/src/instructors`
- `RegisterAuthRoutes(mux *http.ServeMux, ctx *platform.ServerContext)`
  Registers registration, confirmation, login, logout, password recovery, reset,
  and profile password-change routes.
- `type AuthService interface`
  - `CurrentInstructor(r *http.Request) (*Instructor, error)`
  - `RequireConfirmedInstructor(r *http.Request) (*Instructor, error)`

`src/courses/courses.go`
APIs: `RegisterCourseRoutes`, `CourseService`
- Import: `pixelbonus/src/courses`
- `RegisterCourseRoutes(mux *http.ServeMux, ctx *platform.ServerContext)`
  Registers instructor course list, create, and settings routes.
- `type CourseService interface`
  - `ListByInstructor(ctx context.Context, instructorID int64) ([]Course, error)`
  - `Create(ctx context.Context, input CreateCourseInput) (Course, error)`
  - `UpdateSettings(ctx context.Context, input UpdateCourseInput) error`
  - `Get(ctx context.Context, instructorID, courseID int64) (Course, error)`
  - `GetPublic(ctx context.Context, courseID int64) (Course, error)`

`src/coupons/coupons.go`
APIs: `RegisterCouponRoutes`, `CouponService`
- Import: `pixelbonus/src/coupons`
- `RegisterCouponRoutes(mux *http.ServeMux, ctx *platform.ServerContext, courses CourseLookup)`
  Registers token batch creation and secure PDF download routes.
- `type CouponService interface`
  - `CreateBatch(ctx context.Context, input CreateBatchInput) (BatchRequest, error)`
  - `GetDownloadBySecret(ctx context.Context, secret string) (DownloadFile, error)`
  - `LookupToken(ctx context.Context, normalizedCode string) (TokenView, error)`

`src/redemption/redemption.go`
APIs: `RegisterRedemptionRoutes`
- Import: `pixelbonus/src/redemption`
- `RegisterRedemptionRoutes(mux *http.ServeMux, ctx *platform.ServerContext, coupons CouponLookup)`
  Registers QR landing and manual redemption routes.

`src/reporting/reporting.go`
APIs: `RegisterReportingRoutes`, `CalculateGrades`
- Import: `pixelbonus/src/reporting`
- `RegisterReportingRoutes(mux *http.ServeMux, ctx *platform.ServerContext, courses CourseLookup, coupons CouponLookup)`
  Registers instructor grades page, CSV export, and public redemption routes.
- `CalculateGrades(redemptions []StudentRedemptionCount, method GradingMethod) []StudentGrade`
  Computes grades and ranks from redemption counts.

`src/views/i18n.go`
APIs: `Catalog`, `T`
- Import: `pixelbonus/src/views`
- `var Catalog map[string]map[string]string`
  Translation catalog indexed by language and string key.
- `func T(lang, key string) string`
  Returns translated text, falling back to English.

`src/main.go`
APIs: `main`
- Import: executable entry point only.
- `main()`
  Loads configuration, opens database, initializes schema, assembles services,
  registers routes, and starts the HTTPS server.
