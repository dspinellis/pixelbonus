# Global Specification

These requirements apply to all implementation modules.

## Runtime and deployment

- Implementation language: Go 1.15.
- Produce a single static binary.
- Target platform: Debian stable.
- Run without a reverse proxy.
- Support systemd service management.
- TLS certificates are installed separately in Debian-standard locations and
  configured through runtime settings.

## Persistence

- Use SQLite.
- Do not use an ORM.
- Initialize the database schema automatically from `schema.sql`.
- If the implementation needs indexes, constraints, or seed rows for system
  operation, they must be declared in `schema.sql`.

## Security

- Use secure cookie-based sessions for instructors.
- Use CSRF protection on all state-changing instructor and student form
  submissions.
- Hash instructor passwords with `bcrypt` or `argon2`.
- Validate all external input at module boundaries.
- Token identifiers must contain 160 bits of randomness and be stored in a
  normalized uppercase hexadecimal representation.
- Permanent download URLs must use long random secrets and may be reused by
  anyone who possesses them.

## User interface

- HTML title on all pages: `PixelBonus`.
- Header left: word `PixelBonus` linking to the home page.
- Header right:
  - current language with a menu to switch between English and Greek;
  - current logged-in instructor, when present, with links for profile and
    logout.
- Footer:
  - GitHub project link with icon to
    `https://github.com/dspinellis/pixelbonus`;
  - privacy statement suitable for Greece and the EU GDPR context.
- Main palette color `#18bc9c`.
- Secondary palette color `#2c3e50`.
- No external JavaScript frameworks.
- Localization strings must be stored in a way that supports adding more
  languages later.

## URLs and navigation

- Use readable URLs.
- Student QR URLs must use the form `https://pixelbonus.com/r?c=CODE` after
  substituting the configured public base URL.
- Instructor flows must be reachable through stable, descriptive routes.

## Email

- Send mail through the Unix `mail` command.
- Sender address must be `no-reply@pixelbonus.com`.
- Instructor registration and password-reset flows must follow common usability
  practices, including not exposing whether an address exists during password
  reset.

## Testing

- Every exported API defined in `specs/api.md` must be covered by automated
  unit or integration tests.
- Integration tests must cover registration confirmation, login, course
  creation, token batch creation, redemption, grade computation, CSV export,
  and public redemption display.

## Deliverables

- `go.mod`
- `schema.sql`
- `src/main.go`
- `src/...` implementation files
- `tests/...`
- minimal templates and localization assets required by the application
- `pixelbonus.service`
- `README.md`
- `implementation-notes.md`
- `progress.md`
