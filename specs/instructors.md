Source: `src/instructors/auth.go`
APIs: `RegisterAuthRoutes`, `AuthService`

Implement all instructor identity and account-management functionality.

Routes and behavior

- GET/POST instructor registration form collecting full name, unique email, and
  password.
- Email confirmation flow using a random confirmation token emailed to the user.
- GET/POST instructor login with email and password.
- POST instructor logout.
- GET/POST password recovery by email with a usability-oriented response that
  does not reveal whether the email exists.
- GET/POST password reset through emailed token.
- GET/POST profile password change for logged-in instructors.

Security and validation

- Hash passwords before storage.
- Enforce unique email addresses case-insensitively.
- Use expiring confirmation and reset tokens.
- Maintain authenticated sessions with secure cookies.
- Protect all state-changing forms with CSRF checks.

UI

- Render minimal localized templates matching the global shared layout.
