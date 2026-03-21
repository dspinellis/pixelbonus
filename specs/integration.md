# Integration Specification

The system is integrated through `src/app/app.go` and served by `src/main.go`.

## Entry point

- `src/main.go` loads configuration from `config.Load()`.
- `main` constructs the application with `app.New(cfg)`.
- `main` starts the HTTPS server through `App.Run`.

## Dependency graph

- `app` depends on `config`, `store`, `auth`, `course`, `token`, `redeem`,
  `mail`, `pdf`, and `web`.
- `web` depends on `auth`, `course`, `token`, and `redeem` through the APIs
  defined in `specs/api.md`.
- `auth`, `course`, `token`, and `redeem` depend on `store`.
- `token` depends on `pdf` and `mail`.

## Request integration

### Public routes

- `GET /`
  - Renders the home page in the request language or English by default.
- `GET /privacy`
  - Renders the privacy statement.
- `POST /language`
  - Stores the chosen interface language in a cookie and redirects back.
- `GET /redeem`
  - Renders manual coupon redemption form.
- `POST /redeem`
  - Calls `redeem.RedeemByCode`.
- `GET /r?c=CODE`
  - Calls `redeem.CouponLanding`.
- `POST /r`
  - Calls `redeem.RedeemByQR`.
- `GET /courses/{courseID}/public`
  - Calls `course.PublicRedemptionBoard`.

### Instructor authentication routes

- `GET /instructor/login`
- `POST /instructor/login`
  - Calls `auth.Login`, creates a secure cookie-backed session, and redirects
    to the course list.
- `POST /instructor/logout`
  - Calls `auth.Logout`.
- `GET /instructor/register`
- `POST /instructor/register`
  - Calls `auth.RegisterInstructor` and sends confirmation email.
- `GET /instructor/confirm`
  - Calls `auth.ConfirmInstructor`.
- `GET /instructor/forgot-password`
- `POST /instructor/forgot-password`
  - Calls `auth.StartPasswordReset`.
- `GET /instructor/reset-password`
- `POST /instructor/reset-password`
  - Calls `auth.ResetPassword`.
- `GET /instructor/profile`
- `POST /instructor/profile/password`
  - Calls `auth.ChangePassword`.

### Instructor course routes

- `GET /instructor/courses`
  - Calls `course.ListCourses`.
- `GET /instructor/courses/new`
- `POST /instructor/courses`
  - Calls `course.CreateCourse`.
- `GET /instructor/courses/{courseID}/settings`
  - Calls `course.GetCourse`.
- `POST /instructor/courses/{courseID}/settings`
  - Calls `course.UpdateCourse`.
- `GET /instructor/courses/{courseID}/grades`
  - Calls `course.CalculateGrades`.
- `GET /instructor/courses/{courseID}/grades.csv`
  - Calls `course.ExportGradesCSV`.
- `GET /instructor/courses/{courseID}/tokens/new`
- `POST /instructor/courses/{courseID}/tokens`
  - Calls `token.CreateBatch`.
- `GET /downloads/{secret}`
  - Calls `token.ResolveDownload` and serves the generated PDF.

## Batch generation workflow

1. Instructor submits desired token count.
2. `web` validates the request and calls `token.CreateBatch`.
3. `token` creates a batch row and unused token rows through `store`.
4. `token` renders the PDF through `pdf`.
5. `token` creates a permanent download secret through `store`.
6. `token` emails the instructor the download URL through `mail`.
7. The route `/downloads/{secret}` serves the stored PDF file.

## Grading workflow

1. `course.CalculateGrades` loads redemption totals for the course.
2. It orders students by redemption count descending, then computes rank.
3. It derives quartiles over the redemption distribution.
4. It applies the course grading method to obtain an integer grade in the
   range 0 through 10.
5. The resulting rows are exposed in HTML and CSV.

## Language integration

- Request language preference is resolved in this order:
  - explicit language cookie,
  - browser preferred language,
  - English default.
- All templates render text through localization catalogs managed by `web`.

## Error handling

- Invalid student coupon input returns a user-facing error page or inline form
  error without exposing internal details.
- Expired, missing, malformed, or unknown instructor workflow tokens return
  friendly failure pages.
- Internal errors are logged server-side and returned as generic HTTP 500
  responses.
