# Module: csrf

- Target source file path: `src/csrf/service.go`
- Owned APIs:
  - `Issue`
  - `Verify`
- Internal logic:
  - generate HMAC-protected token including session id and timestamp
  - verify token age and signature
- Dependencies:
  - none beyond `design/api.md`
