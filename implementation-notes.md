# Implementation Notes

- MySQL driver: `github.com/go-sql-driver/mysql` v1.7.1 supports Go 1.15. Domain queries use the table and column names in `old-schema.sql`; schema initialization is additive. SQLite is limited to the unit-test fixture.
- Session cookies use random server-side session identifiers stored in MySQL. Session ids are opaque and rotated by expiry updates.
- CSRF protection is session-or-visitor based. Anonymous users receive a visitor cookie so student redemption forms can also carry CSRF tokens.
- Translation catalogs are loaded from `src/translations/en.json` and `src/translations/el.json`.
- Markdown rendering for `privacy_policy.md` is intentionally minimal and supports headings, paragraphs, and bullet lists.
- Quartiles use nearest-rank calculation. Ranks use standard competition ranking.
- Token generation jobs are asynchronous goroutines resumed from persisted `pending` rows on startup.
- Download links use the generated PDF basename as the permanent nonce.
- The privacy policy placeholders from [privacy_policy.md](/home/dds/src/pixelbonus2/privacy_policy.md) are intentionally preserved and must be replaced during deployment.
