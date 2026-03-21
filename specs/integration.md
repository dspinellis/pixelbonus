Entry point

- The executable entry point is `src/main.go`.
- At startup it loads configuration, opens SQLite, initializes `schema.sql`,
  starts background workers, registers all routes, and serves HTTPS traffic.

Route and module interactions

1. Shared web infrastructure
   - `platform.NewServerContext` creates dependencies shared by all route
     modules.
   - `platform.RegisterSharedRoutes` provides home page, language switching, and
     privacy page.

2. Instructor authentication
   - `instructors.RegisterAuthRoutes` uses `platform.ServerContext` for template
     rendering, sessions, CSRF, localization, mail sending, and token storage.
   - Authenticated instructor identity is exposed through the auth service and
     request context for downstream modules.

3. Course management
   - `courses.RegisterCourseRoutes` is wrapped by instructor authentication.
   - Course handlers persist and fetch only through the course module API.

4. Coupon generation
   - `coupons.RegisterCouponRoutes` requires authenticated instructors.
   - Coupon generation validates course ownership through the course lookup API.
   - Successful batch creation schedules PDF rendering and email sending.
   - Download links are resolved only through the coupon service.

5. Redemption
   - `redemption.RegisterRedemptionRoutes` uses the coupon lookup API to resolve
     submitted token codes and course metadata.
   - Redemptions mark a token as used exactly once and store the student id.

6. Reporting
   - `reporting.RegisterReportingRoutes` requires authenticated instructors for
     grades and CSV export, and allows public access for redemption counts.
   - Reporting reads owned course metadata through `CourseService.Get`, public
     course metadata through `CourseService.GetPublic`, token/redemption data
     through the coupon API, then applies `CalculateGrades`.

Shared data contracts

- Grading is computed from per-student redemption counts within a single course.
- Public reporting exposes only student id and redemption count.
- Instructor reporting exposes student id, redemption count, grade, and rank.
- Student identifiers are scoped only by course data usage; there is no global
  student account.
