Use case: Instructor registers and confirms account
- Call `platform.NewServerContext`
- Call `instructors.RegisterAuthRoutes`
- Submit registration form
- Follow emailed confirmation link handled by auth routes
- Result: achievable with current APIs

Use case: Instructor logs in and changes password
- Call `platform.NewServerContext`
- Call `instructors.RegisterAuthRoutes`
- Submit login form
- Access profile password-change route
- Result: achievable with current APIs

Use case: Instructor creates and configures a course
- Authenticate through auth routes
- Call `courses.RegisterCourseRoutes`
- Use `CourseService.Create`
- Use `CourseService.UpdateSettings`
- Result: achievable with current APIs

Use case: Instructor requests a QR coupon batch and downloads the generated PDF
- Authenticate through auth routes
- Resolve course ownership with `CourseService.Get`
- Submit batch request through `coupons.RegisterCouponRoutes`
- Use `CouponService.CreateBatch`
- Later resolve file through `CouponService.GetDownloadBySecret`
- Result: achievable with current APIs

Use case: Student redeems a coupon from a QR link
- Resolve token through `CouponService.LookupToken`
- Submit redemption through `redemption.RegisterRedemptionRoutes`
- Result: achievable with current APIs

Use case: Student redeems a coupon by entering its identifier
- Normalize code and resolve token through `CouponService.LookupToken`
- Submit redemption through `redemption.RegisterRedemptionRoutes`
- Result: achievable with current APIs

Use case: Instructor views grades and downloads CSV
- Authenticate through auth routes
- Resolve owned course through `CourseService.Get`
- Register reporting routes
- Read course/token data through reporting dependencies
- Apply `CalculateGrades`
- Result: achievable with current APIs

Use case: Public visitor views course redemption counts
- Resolve course metadata through `CourseService.GetPublic`
- Read redemption counts through reporting dependencies
- Render public page
- Result: achievable with revised APIs
