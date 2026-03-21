Source: `src/config/config.go`
APIs: `Load`

# Runtime configuration

Implement runtime configuration loading for the application.

## Responsibilities

- Read configuration from environment variables with Debian-friendly defaults.
- Return a typed configuration value used by `app.New` and `main`.

## Required configuration fields

- HTTPS listen address.
- TLS certificate path.
- TLS private key path.
- SQLite database path.
- Path to `schema.sql`.
- Public base URL used in generated links.
- Session cookie name and security settings.
- Mail command path.
- Directory for generated PDFs.
- Paths for templates and any static assets if required by the web module.

## Behavior

- Reject invalid or incomplete mandatory settings with a descriptive error.
- Normalize filesystem paths where useful.
- Defaults must support local development where practical without weakening
  secure-cookie behavior in production configuration.
