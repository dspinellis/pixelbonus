Global requirements applicable to all modules.

Implementation language and packaging

- Use Go 1.15 conventions.
- Produce a single deployable binary.
- Organize source code under `src/` and tests under `tests/`.
- Avoid non-standard dependencies except where strictly needed for SQLite
  access, QR generation, PDF generation, password hashing, and CSRF/session
  support that would otherwise be impractical with the standard library.

Persistence and schema

- Use SQLite directly without an ORM.
- Initialize the database automatically from `schema.sql`.
- All database mutations must be parameterized and validated.

Security and privacy

- Use secure password hashing with bcrypt or argon2.
- Use secure cookies for login state.
- Provide CSRF protection for all state-changing form submissions.
- Validate all input lengths and formats.
- Implement email confirmation and password reset tokens as single-purpose,
  random, high-entropy values with expiry.
- Provide an EU/Greece-oriented privacy statement page.

Localization and UI

- Support English and Greek, defaulting to the browser-preferred language and
  falling back to English.
- Store localizable strings in a dedicated translation catalog.
- All pages share a layout with:
  - title `PixelBonus`
  - home link on the left
  - language selector on the right
  - current user menu on the right when logged in
  - footer links to GitHub and privacy statement
- Use the specified palette colors `#18bc9c` and `#2c3e50`.

Operational constraints

- Designed for Debian stable and systemd operation.
- No reverse proxy requirement.
- Expect Let's Encrypt certificate files to already exist in Debian-standard
  filesystem locations.
- Mail is sent through the Unix `mail` command with sender
  `no-reply@pixelbonus.com`.

Testing

- Every exported module API must have automated tests.
- Add integration tests for end-to-end flows after module implementation.
- Ignore editor swap files.
