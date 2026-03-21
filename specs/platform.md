Source: `src/platform/config.go`, `src/platform/db.go`, `src/platform/httpctx.go`
APIs: `LoadConfig`, `OpenDatabase`, `InitializeSchema`, `WithTx`,
`NewServerContext`, `RegisterSharedRoutes`, `RequireInstructor`

Implement shared platform services for the application.

Responsibilities

- Read configuration needed for database path, TLS certificate/key paths, HTTP
  listen address, base URL, mail command path, generated file directory, and
  session security settings.
- Open the SQLite database and initialize it by executing `schema.sql` exactly
  as part of startup if the schema objects do not yet exist.
- Provide the shared HTTP server context:
  - template rendering
  - shared page layout data
  - session access
  - CSRF protection
  - localization resolution and language switch route
  - privacy statement route
  - home page route
- Determine preferred language from explicit selection first, then request
  headers, then English default.
- Expose a middleware that rejects unauthenticated instructor access and
  redirects to the instructor login page.

Home page requirements

- Host under the application root.
- Present the student and instructor halves described in the original
  specifications.
- Use readable URLs for links to instructor and student flows.
