Source: `src/main.go`
APIs: `main`

# Executable entry point

Implement the program entry point.

## Responsibilities

- Load runtime configuration through `config.Load`.
- Construct the application through `app.New`.
- Start the server through `App.Run`.
- Exit with a non-zero status on startup failure.

## Behavior

- Log startup errors to standard error.
- Use context and signal handling suitable for clean shutdown under systemd
  and during integration tests.
