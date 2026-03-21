Source: `src/redemption/redemption.go`
APIs: `RegisterRedemptionRoutes`

Implement all student-facing redemption flows without registration.

Routes and behavior

- GET redemption landing page by QR parameterized link that resolves a coupon
  and displays:
  - course name as read-only
  - coupon code as read-only
  - student id input
  - redeem button
- GET/POST manual redemption page allowing coupon-code entry in hexadecimal with
  optional spaces, uppercase normalization, and client/server-side rejection of
  non-hex input.
- POST redemption submission that produces exactly one of:
  - success message including coupon number, course name, and student id
  - already redeemed message including coupon number and course name
  - invalid coupon identifier message including coupon number

Data rules

- Students do not authenticate.
- A token may be redeemed at most once.
- Student identifiers are arbitrary strings up to 20 characters.
