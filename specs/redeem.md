Source: `src/redeem/redeem.go`
APIs: `NewService`, `CouponLanding`, `RedeemByCode`, `RedeemByQR`,
`PublicCourseBoard`

# Student redemption

Implement student-facing coupon lookup and redemption.

## Data handled by this module

- Student redemption attempts and successful redemptions.
- Token status transitions from unused to redeemed.

## Functional requirements

- Support redeeming a coupon through a QR landing page and through manual code
  entry.
- Students never register or authenticate.
- A student identifier is an arbitrary string up to 20 characters and has no
  cross-course linkage.
- A student may redeem any number of coupons in the same course as long as
  each coupon itself has not already been redeemed.

## QR landing

- `CouponLanding` accepts a normalized token code from the query string.
- It returns:
  - course name,
  - normalized coupon code,
  - formatted coupon code for display.

## Redemption behavior

- Manual redemption accepts input containing spaces between hex groups and
  normalizes it before lookup.
- Student id is required, trimmed, and limited to 20 characters.
- If the token exists and is unused, redemption records:
  - token id,
  - course id,
  - student id,
  - redemption timestamp,
  and marks the token as redeemed atomically.
- If the token exists and is already redeemed, return the
  `already redeemed` result.
- If the token does not exist or fails validation, return the
  `invalid coupon identifier` result.

## Outputs

- Successful redemption result includes:
  - printable coupon code,
  - course name,
  - student id.
- Already-redeemed result includes:
  - printable coupon code,
  - course name.
- Invalid result includes the submitted printable coupon code when possible.
