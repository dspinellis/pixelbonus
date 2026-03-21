# PixelBonus

PixelBonus is a Go 1.15 web application that lets instructors generate printable QR coupon sheets and lets students redeem participation coupons without creating accounts.

## Build

```sh
go build -o pixelbonus ./src
```

## Run

Set optional configuration through environment variables:

- `PIXELBONUS_ADDR`
- `PIXELBONUS_DB`
- `PIXELBONUS_SCHEMA`
- `PIXELBONUS_TEMPLATES`
- `PIXELBONUS_ASSETS`
- `PIXELBONUS_PRIVACY`
- `PIXELBONUS_PDF_DIR`
- `PIXELBONUS_BASE_URL`
- `PIXELBONUS_TLS_CERT`
- `PIXELBONUS_TLS_KEY`
- `PIXELBONUS_MAIL_CMD`

Then run:

```sh
./pixelbonus
```

## Test

```sh
go test ./...
```

## Deploy

1. Install the binary, `src/templates`, `src/assets`, `privacy_policy.md`, and `design/schema.sql`.
2. Ensure the letsencrypt certificate files exist at `/etc/letsencrypt/live/pixelbonus.com/fullchain.pem` and `/etc/letsencrypt/live/pixelbonus.com/privkey.pem`.
3. Install `src/pixelbonus.service` under `/etc/systemd/system/pixelbonus.service`.
4. Run `systemctl enable --now pixelbonus`.

## Use

1. Register an instructor account.
2. Verify the account through the email link.
3. Log in and create a course.
4. Request coupon generation in multiples of 20.
5. Wait for the email containing the PDF download link.
6. Print and distribute the coupons.
7. Students redeem through the QR code or the manual redemption page.

## Monitor

- Review the `systemd` unit logs with `journalctl -u pixelbonus`.
- Inspect the SQLite database file for job failures.

## Maintenance

Delete old generated PDFs with cron, for example:

```cron
0 3 * * * find /var/tmp/pixelbonus -type f -name '*.pdf' -mtime +30 -delete
```
