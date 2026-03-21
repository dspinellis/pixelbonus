Source: `src/auth/auth.go`
APIs: `NewService`, `RegisterInstructor`, `ConfirmInstructor`, `Login`,
`Logout`, `StartPasswordReset`, `ResetPassword`, `ChangePassword`,
`ValidateSession`, `NewCSRFToken`, `VerifyCSRFToken`

# Authentication and instructor identity

Implement instructor-only identity management.

## Data handled by this module

- Instructors with:
  - numeric id,
  - full name,
  - unique email address,
  - password hash,
  - preferred language,
  - confirmation state,
  - creation and update timestamps.
- Confirmation tokens:
  - random secret,
  - instructor id,
  - creation timestamp,
  - consumed timestamp.
- Password reset tokens:
  - random secret,
  - instructor id,
  - creation timestamp,
  - consumed timestamp.
- Sessions:
  - random session id,
  - instructor id,
  - CSRF secret,
  - creation timestamp,
  - last-seen timestamp.

## Functional requirements

- Register an instructor from full name, email, password, and interface
  language.
- Enforce unique email addresses case-insensitively.
- Create unconfirmed accounts and a confirmation token suitable for inclusion
  in an emailed link.
- Confirm an instructor account by token exactly once.
- Reject login for unknown, unconfirmed, or password-mismatched instructors.
- Start password reset without revealing whether an email exists.
- Reset password by one-time token.
- Allow logged-in instructors to change password by providing the current
  password.
- Create and validate secure session identifiers for cookie-based login.
- Generate and verify CSRF tokens bound to the session.

## Validation

- Full name is required and limited to 120 characters.
- Email must be syntactically valid and normalized to lower case.
- Password must be at least 10 characters.
- Tokens and session ids must be random, opaque strings.

## Security requirements

- Passwords must be hashed with `bcrypt` or `argon2`.
- Sessions must be invalidated on logout.
- Confirmation and reset tokens must be single use.
- Password reset and confirmation operations must be idempotent in user-facing
  behavior and safe under repeated requests.

## Outputs

- Registration returns the instructor identity and the confirmation token for
  later email delivery by the caller.
- Login returns the instructor identity and session id.
- Password reset start returns a result that allows the caller to send a reset
  email when the account exists.
