# Module: logging

- Target source file path: `src/logging/logger.go`
- Owned APIs:
  - `New`
  - `Log`
- Internal logic:
  - open append-only log file
  - serialize writes with mutex
  - format deterministic key ordering for fields
- Dependencies:
  - `config.Load`
