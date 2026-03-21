# Use Cases

## Instructor registration and verification

1. Instructor opens `GET /instructor/register`.
2. Instructor submits name, email, and password to `POST /instructor/register`.
3. System validates input, creates an unverified instructor, stores a 24-hour verification token, and sends an email with `GET /instructor/verify?token=...`.
4. Instructor opens the verification URL.
5. System marks the instructor verified and renders a confirmation page.

## Instructor login and logout

1. Instructor opens `GET /instructor/login`.
2. Instructor submits credentials to `POST /instructor/login`.
3. System authenticates the verified instructor, creates a session and CSRF token, sets the secure cookie, and redirects to `GET /instructor/courses`.
4. Instructor logs out through `POST /instructor/logout`.
5. System deletes the session and clears the cookie.

## Password recovery

1. Instructor opens `GET /instructor/password/forgot`.
2. Instructor submits email to `POST /instructor/password/forgot`.
3. System always renders the same acknowledgment message.
4. For an existing verified account, system stores a one-hour reset token and emails `GET /instructor/password/reset?token=...`.
5. Instructor submits new password to `POST /instructor/password/reset`.
6. System updates the password and consumes the token.

## Profile password change

1. Logged-in instructor opens `GET /instructor/profile`.
2. Instructor submits a new password to `POST /instructor/profile/password`.
3. System validates the password, updates the hash, and renders a success message.

## Course management

1. Logged-in instructor opens `GET /instructor/courses`.
2. System lists owned courses ordered by most recent creation date first.
3. Instructor opens `GET /instructor/courses/new`, submits `POST /instructor/courses`, and the course is created.
4. Instructor opens `GET /instructor/courses/{courseID}/settings`, submits `POST /instructor/courses/{courseID}/settings`, and the course is updated.

## Coupon generation

1. Logged-in instructor opens `GET /instructor/courses/{courseID}/coupons/new`.
2. Instructor submits a quantity that is a positive multiple of 20 to `POST /instructor/courses/{courseID}/coupons`.
3. System inserts a queued generation job and immediately renders an acknowledgment.
4. Background worker claims the job, generates 160-bit coupon tokens, writes the PDF to `/var/tmp/pixelbonus/{nonce}.pdf`, stores the file metadata, and emails the download link `GET /downloads/{nonce}.pdf`.

## Grade viewing and download

1. Logged-in instructor opens `GET /instructor/courses/{courseID}/grades`.
2. System computes redemption counts, quartiles, grades, and ranks, then renders an ordered table by student identifier.
3. Instructor opens `GET /instructor/courses/{courseID}/grades.csv`.
4. System returns the same data in CSV format with the header `student_id,redemptions,grade,rank`.

## Public redemption view

1. Any user opens `GET /courses/{courseID}/redemptions`.
2. System renders the course name and ordered student redemption counts.

## Student redemption by QR link

1. Student scans a QR code containing `GET /r?c={code}`.
2. System resolves the coupon and renders the redemption form with read-only course name and coupon code.
3. Student submits `POST /redeem` with the coupon code and student identifier.
4. System atomically redeems the coupon or reports already redeemed or invalid coupon.

## Student redemption by manual code

1. Student opens `GET /redeem`.
2. System renders a form for coupon code and student identifier.
3. Student submits `POST /redeem`.
4. System normalizes the code by removing spaces and uppercasing hexadecimal characters, rejects non-hex input, and returns the required outcome message.
