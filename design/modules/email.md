# Module Specification: email

- Target source file path: `src/pixelbonus/email/sender.go`
- Owned APIs:
  - `NewSender`
  - `Sender.Send`
- Internal logic:
  - Define `Sender` as a lightweight struct holding command path and sender address.
  - Execute the Unix `mail` command with subject and sender flags.
  - Feed the email body on standard input.
- Dependencies ONLY via `design/api.md`:
  - none
