# Use Cases

## UC1: Home Page

1. Anonymous user requests `/`.
2. System resolves language.
3. System renders localized home page with student and instructor entry points.

## UC2: Instructor Registration

1. Instructor opens `/instructor/register`.
2. Instructor submits full name, email, and password.
3. System validates input and uniqueness of email.
4. System creates an unverified instructor and a 24-hour verification nonce.
5. System sends an email containing `/instructor/register/confirm?token=...`.
6. Instructor opens the link.
7. System marks email as verified and renders a success page with a link to login.

## UC3: Instructor Login and Logout

1. Instructor opens `/instructor/login`.
2. Instructor submits email and password.
3. System verifies password and that email verification is complete.
4. System issues a secure session cookie.
5. Instructor uses the profile menu to logout.
6. System invalidates the session and clears the cookie.

## UC4: Password Recovery

1. Instructor opens `/instructor/password/forgot`.
2. Instructor submits email.
3. System always renders the same acknowledgement page.
4. If the account exists, system creates a 1-hour reset nonce and emails the reset link.
5. Instructor opens `/instructor/password/reset?token=...`.
6. Instructor sets a new password.
7. System invalidates the token and updates the password hash.

## UC5: Create and Edit Course

1. Authenticated instructor opens `/instructor/courses`.
2. System lists courses newest first.
3. Instructor opens `/instructor/courses/new`.
4. Instructor submits course name and grading method.
5. System creates the course and a public nonce.
6. Instructor may later open `/instructor/courses/{courseID}/settings` and save changes.

## UC6: Generate Coupon Batch

1. Authenticated instructor opens the token generation form for a course.
2. Instructor submits a valid multiple-of-20 count.
3. System creates a token set row and token rows in one transaction.
4. System immediately acknowledges that notification will follow.
5. Background worker generates the PDF in `/var/tmp/pixelbonus/{fileNonce}.pdf`.
6. Background worker sends an email containing `/download/{fileNonce}`.

## UC7: Redeem Via QR Link

1. Student opens `/r?c={encodedToken}`.
2. System decodes token and finds associated course.
3. System renders the redemption form with read-only course name and coupon code.
4. Student submits student identifier.
5. System atomically redeems the token or returns already-used / invalid status.

## UC8: Redeem Via Manual Code

1. Student opens `/redeem`.
2. Student types a coupon code and student identifier.
3. System normalizes the code by removing spaces and uppercasing.
4. System atomically redeems the token or returns already-used / invalid status.

## UC9: View Grades

1. Authenticated instructor opens `/instructor/courses/{courseID}/grades`.
2. System computes redemption counts, ranks, quartiles, and grades.
3. System renders results ordered by student identifier.
4. Instructor may download `/instructor/courses/{courseID}/grades.csv` with the same data.

## UC10: View Public Redemptions

1. Any visitor opens `/public/{publicNonce}`.
2. System resolves the course by public nonce.
3. System renders student identifier and redemption count ordered by student identifier.

## UC11: Privacy Policy

1. Any visitor opens `/privacy`.
2. System converts `privacy_policy.md` to HTML.
3. System renders it inside the standard localized layout.
