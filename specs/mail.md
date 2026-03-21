Source: `src/mail/mail.go`
APIs: `NewSender`, `Send`

# Email delivery

Implement email delivery through the Unix `mail` command.

## Responsibilities

- Send plain-text transactional messages.
- Set the sender address to `no-reply@pixelbonus.com`.
- Support the following messages:
  - instructor registration confirmation,
  - password reset,
  - token batch download notification.

## Behavior

- The API returns an error if the command cannot be started or exits with a
  failure status.
- Message composition is handled by the caller; this module only delivers the
  provided recipient, subject, and body.
