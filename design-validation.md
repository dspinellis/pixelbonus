# Design Validation

This document validates that each use case can be implemented using only the
declared APIs in `specs/api.md`.

## Home page browsing

- `web.NewServer`
- `Server.Routes`

## Language selection

- `web.SetLanguage`
- `web.Routes`

## Instructor registration and confirmation

- `auth.RegisterInstructor`
- `mail.Sender.Send`
- `auth.ConfirmInstructor`

## Instructor login/logout

- `auth.Login`
- `auth.Logout`
- `auth.ValidateSession`
- `web.RequireInstructor`

## Password reset

- `auth.StartPasswordReset`
- `mail.Sender.Send`
- `auth.ResetPassword`

## Profile password change

- `auth.ChangePassword`

## Course creation and listing

- `course.CreateCourse`
- `course.ListCourses`

## Course settings update

- `course.GetCourse`
- `course.UpdateCourse`

## Token batch generation and instructor notification

- `course.GetCourse`
- `token.CreateBatch`
- `pdf.Generator.RenderCoupons`
- `mail.Sender.Send`

## Secure download of generated PDF

- `token.ResolveDownload`

## Manual coupon redemption

- `token.NormalizeCode`
- `redeem.RedeemByCode`

## QR-link coupon redemption

- `redeem.CouponLanding`
- `redeem.RedeemByQR`

## Instructor grade viewing

- `course.CalculateGrades`

## Instructor CSV export

- `course.ExportGradesCSV`

## Public redemption board viewing

- `course.PublicRedemptionBoard`

## Coverage conclusion

All listed use cases map to declared package APIs. No use case requires direct
cross-module access outside `specs/api.md`.
