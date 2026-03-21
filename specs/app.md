Source: `src/app/app.go`
APIs: `New`, `Handler`, `Run`

# Application assembly

Implement startup wiring and runtime orchestration.

## Responsibilities

- Load or receive runtime configuration.
- Open the store and initialize the schema.
- Construct service modules and the web server with their dependencies.
- Expose the composed HTTP handler for tests.
- Run the HTTPS server with the configured address and certificate paths.

## Runtime behavior

- Startup failure in any dependency must return an error.
- `Run` must support context-based shutdown for tests and system integration.
- Logging should go to standard error so the service is systemd-friendly.
