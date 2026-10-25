# Module: mail

- Target source file path: `src/mail/service.go`
- Owned APIs:
  - `Send`
- Internal logic:
  - validate recipient
  - invoke Unix `mail`
  - set sender envelope/header to `no-reply@pixelbonus.com`
  - capture exit status
- Dependencies:
  - `logging.Log`
