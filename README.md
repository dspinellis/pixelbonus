# PixelBonus

PixelBonus is a Go 1.15 web application for instructor-managed classroom participation using redeemable QR coupons.
See it in action and use it in your class through [pixelbonus.com](https://www.pixelbonus.com/).

The source code in this repository was developed using OpenAI's Codex to replace a
[legacy PHP/Symfony implementation](https://github.com/dspinellis/pixelbonus-php)
following the publication of two critical PHP vulnerabilities
([CVE-2026-109618](https://www.cve.org/CVERecord?id=CVE-2026-109618) and
[CVE-2026-109619](https://www.cve.org/CVERecord?id=CVE-2026-109619))
and the lack of packaged updates for the PHP version the legacy code was implemented in.

## Build

Run:

```bash
GOPATH=/tmp/go GOMODCACHE=/tmp/go/pkg/mod GO111MODULE=on go build ./...
```

## Test

Run from the repository root:

```bash
GOPATH=/tmp/go GOMODCACHE=/tmp/go/pkg/mod GO111MODULE=on go test ./...
```

Tests live together in `tests/`, so the `pixelbonus/src/...` packages report `[no test files]`. The final `ok pixelbonus/tests` line confirms that the suite passed; `(cached)` means Go reused a previous successful result.

To show individual tests and force a fresh run of the test suite:

```bash
GOPATH=/tmp/go GOMODCACHE=/tmp/go/pkg/mod GO111MODULE=on go test -v -count=1 ./tests
```

MySQL-specific tests require `PIXELBONUS_TEST_MYSQL_DSN`; see below for server setup. Without it, those tests skip while the remaining tests run using SQLite.

## MySQL setup

The application uses MySQL 5.7+ (also tested with MariaDB 10.5) and the domain tables from [legacy schema](design/legacy-schema.md). SQLite is used only for fast unit tests.

Set the connection using `db_dsn` in the config file, or the overriding `PIXELBONUS_DB_DSN` environment variable:

```ini
db_dsn=pixelbonus:password@tcp(127.0.0.1:3306)/pixelbonus
```

Unix sockets also work: `pixelbonus:password@unix(/run/mysqld/mysqld.sock)/pixelbonus`. For remote servers configure the driver's `tls` DSN option as appropriate. DSN parameters follow [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql/tree/v1.7.1#dsn-data-source-name). The application always reads DATETIME values as strings and disables multi-statement execution.

For an **existing legacy database**, select it in the DSN and start normally. Startup applies the additive [application schema](design/schema.sql): `CREATE TABLE IF NOT EXISTS` statements only, with no copying, dropping, renaming, or altering of existing tables. Do not import [legacy schema](design/legacy-schema.md) over an existing database: that dump includes destructive DROP statements. The application account needs SELECT, INSERT, UPDATE, DELETE, CREATE, and REFERENCES privileges on the selected database.

For a **fresh database**, build the executable and initialize it:

```bash
go build -o pixelbonus ./src
PIXELBONUS_CONFIG=/etc/pixelbonus.conf ./pixelbonus -init-db
```

This creates the selected database if missing, creates its application tables, and exits without starting HTTP or PDF jobs. The account must have permission to create the selected database. Alternatively, have an administrator create an empty database, then start the app normally to create its tables. Initialization can safely be repeated.

Compatibility details:

- Data stays in `Users`, `Course`, `QrSet`, `QrRequest`, `QrCode`, and `Redemption`. `Tag` and `qrset_tags` are preserved and included for fresh installs. Symfony ACL/logging/translation tables are unused.
- `QrSet.user_id` references `Course.id`; `Course.user_id` references `Users.id`. Existing IDs, coupon codes, `Course.hashedUrl`, account status flags, and redemption history are preserved.
- Sessions and authentication tokens use separate tables; `pixelbonus_course_settings` stores per-course grading choices and `pixelbonus_pdf_jobs` stores PDF file metadata. Old requests have no Go PDF metadata, so they are not restarted or offered as Go downloads. New token generation writes the legacy QR tables transactionally.
- Existing enabled accounts can log in using bcrypt (including PHP `$2y$`) or Symfony's default salted SHA-512 encoder (5000 iterations, base64 or hex). This follows [Symfony's encoder implementation](https://github.com/symfony/security-core/blob/2.8/Encoder/MessageDigestPasswordEncoder.php). The dump does not specify the historical encoder configuration; accounts using other encoders need a password reset. New and reset passwords use bcrypt. Concurrent PHP authentication would require its encoder configuration to accept these hashes. PHP sessions and outstanding PHP confirmation/reset tokens are not imported.
- Legacy courses use the owner's `preferred_grading_model` when it matches a supported model, otherwise `0=0,>=Q2=10`. Changing a course stores its grading method separately. Legacy user multiplier/minimum/maximum settings are preserved; the current grading engine continues to produce grades from 0 to 10.
- Legacy DATETIME values are interpreted as UTC; new DATETIME writes use UTC.

Run all tests against a real server (the account needs CREATE/DROP DATABASE rights):

```bash
PIXELBONUS_TEST_MYSQL_DSN='tester:password@tcp(127.0.0.1:3306)/' go test ./...
```

Tests create uniquely named temporary databases and drop only those databases afterward. They cover fresh initialization, repeated initialization over the actual legacy table definitions, existing records, concurrent redemption, account flows, grading, and PDF job persistence. Without this variable, MySQL-specific tests skip and the remaining tests use the SQLite fixture.

## Continuous Integration

[GitHub Actions](.github/workflows/ci.yml) builds, vets, and tests the project on
pushes and pull requests, using Go 1.15 and the current stable Go release. Each
version runs the suite with SQLite and a disposable MySQL 8.0 service. No repository
secrets or production database are required. The workflow can also be run manually.

## Run

1. Install TLS files at `/etc/letsencrypt/live/pixelbonus.com/fullchain.pem` and `/etc/letsencrypt/live/pixelbonus.com/privkey.pem`.
2. Optionally create `/etc/pixelbonus.conf`:

```ini
port=443
log_dir=/var/log/pixelbonus
template_dir=src/templates
base_url=https://pixelbonus.com
db_dsn=pixelbonus:password@tcp(127.0.0.1:3306)/pixelbonus
```

3. Start the binary from the repository root or deployed working directory:

```bash
PIXELBONUS_CONFIG=/etc/pixelbonus.conf ./pixelbonus
```

The application adds missing MySQL tables automatically from [design/schema.sql](design/schema.sql). Run from a directory containing that schema and the configured templates/translations.

For a trial run as an ordinary user, use `PIXELBONUS_CONFIG=pixelbonus-try.conf ./pixelbonus`
and open `https://www.pixelbonus.com:8443/` (the hostname must resolve to the machine
running PixelBonus). This configuration uses port 8443 because binding port 443 on
Linux normally requires root or `CAP_NET_BIND_SERVICE`. The TLS certificate and key
listed above must still be readable by the application user. `base_url` controls
generated links; `port` controls the listening port. HTTPS startup failures are
printed to standard error and cause the process to exit.

## Deploy with systemd

The supplied [service unit](deploy/pixelbonus.service) runs as a dedicated
`pixelbonus` user from `/opt/pixelbonus`, reads `/etc/pixelbonus.conf`, and starts
at boot. It grants `CAP_NET_BIND_SERVICE` so the application can bind HTTPS port
443 without running as root. It restarts after failures with a five-second delay.

Run the following commands from the repository root on the deployment machine.
The account and package commands below are for Debian/Ubuntu. Install and configure
MySQL/MariaDB as described above, and provision a TLS certificate covering your
public hostname at `/etc/letsencrypt/live/pixelbonus.com/`. The application currently
uses that certificate path regardless of `base_url`.

1. Install the mail client and ACL tools. The local mail transport must already
   be configured to deliver mail; PixelBonus invokes the `mail` command.

   ```bash
   sudo apt-get install mailutils acl
   ```

2. Build and install the application and its runtime files:

   ```bash
   GOPATH=/tmp/go GOMODCACHE=/tmp/go/pkg/mod GO111MODULE=on go build -o pixelbonus ./src
   id pixelbonus >/dev/null 2>&1 || sudo useradd --system --user-group --home-dir /nonexistent --no-create-home --shell /usr/sbin/nologin pixelbonus
   sudo install -d -m 0755 /opt/pixelbonus /opt/pixelbonus/src /opt/pixelbonus/design
   sudo install -m 0755 pixelbonus /opt/pixelbonus/pixelbonus
   sudo install -m 0644 design/schema.sql /opt/pixelbonus/design/schema.sql
   sudo cp -R src/templates src/translations /opt/pixelbonus/src/
   sudo install -m 0644 privacy_policy.md /opt/pixelbonus/privacy_policy.md
   sudo install -d -o pixelbonus -g pixelbonus -m 0750 /var/log/pixelbonus /var/tmp/pixelbonus
   ```

3. Install the [configuration template](deploy/pixelbonus.conf) and edit it:

   ```bash
   sudo install -o root -g pixelbonus -m 0640 deploy/pixelbonus.conf /etc/pixelbonus.conf
   sudoedit /etc/pixelbonus.conf
   ```

   Set `db_dsn` to your actual database credentials and `base_url` to the public
   HTTPS address, including the port if you change `port` from 443. Keep
   `log_dir=/var/log/pixelbonus`. The service user must be able to connect to MySQL;
   it needs the table privileges listed in the MySQL setup section. For a fresh
   database, initialize it before starting the service:

   ```bash
   cd /opt/pixelbonus
   sudo -u pixelbonus env PIXELBONUS_CONFIG=/etc/pixelbonus.conf ./pixelbonus -init-db
   cd -
   ```

   Initialization requires permission to create the selected database. For an
   existing database, start the service normally; it only adds missing tables.

4. Install the unit and the [Certbot deploy hook](deploy/pixelbonus-certbot-hook.sh).
   The hook grants the service user read access to the certificate and private
   key, including directory traversal, and restarts an active service after
   certificate renewal. Run it once to grant access to the current certificate:

   ```bash
   sudo install -m 0644 deploy/pixelbonus.service /etc/systemd/system/pixelbonus.service
   sudo systemctl daemon-reload
   sudo install -d -m 0755 /etc/letsencrypt/renewal-hooks/deploy
   sudo install -m 0755 deploy/pixelbonus-certbot-hook.sh /etc/letsencrypt/renewal-hooks/deploy/pixelbonus.sh
   sudo /etc/letsencrypt/renewal-hooks/deploy/pixelbonus.sh
   ```

   If you use a different certificate manager, run the same hook after it updates
   these TLS files. Keep the private key restricted to its existing owner and the
   service user; the hook uses ACLs rather than making it publicly readable.

5. Stop any manually launched PixelBonus process that occupies the configured
   port, allow inbound HTTPS through your firewall, then enable and start:

   ```bash
   sudo systemctl enable --now pixelbonus.service
   sudo systemctl status pixelbonus.service
   sudo journalctl -u pixelbonus.service -n 50 --no-pager
   sudo tail -f /var/log/pixelbonus/pixelbonus.log
   ```

   Open the configured `base_url`. Startup errors also appear in the systemd
   journal, including failures to open the configuration, log, database, port, or
   TLS files. Local MySQL/MariaDB startup is ordered before PixelBonus when their
   services are started in the same boot transaction; enable the database service
   separately if it runs on this machine.

For updates, stop the service, repeat the binary and runtime-file installation
from step 2, and start it again. Preserve your existing `/etc/pixelbonus.conf`
instead of reinstalling the template. Restart after configuration, template, or
translation changes:

```bash
sudo systemctl restart pixelbonus.service
```

PDFs stay in `/var/tmp/pixelbonus` across service restarts. To delete PDFs older
than 30 days, install the following entry using `sudo crontab -e`:

```cron
0 3 * * * find /var/tmp/pixelbonus -type f -name '*.pdf' -mtime +30 -delete
```

## Use

- Home page: `/`
- Instructor login: `/instructor/login`
- Instructor registration: `/instructor/register`
- Manual coupon redemption: `/redeem`
- Public course redemptions: `/public/{nonce}` (legacy alias: `/course_overview/{nonce}`)

## Monitoring

- Application events are appended to `pixelbonus.log` under the configured log directory.
- Startup, shutdown, startup failures, and HTTP requests (method, path, status, and duration) are logged. Request query strings are omitted. Relative `log_dir` paths are resolved from the working directory; the trial configuration writes to `log/pixelbonus.log`.
- systemd restart policy is `on-failure`.

## Privacy Policy Configuration

Set `privacy_policy_urls.en` and `privacy_policy_urls.el` in the configuration
file to the English and Greek policy URLs. The footer uses the URL for the
selected language. Both default to `/privacy`, which serves the local policy.
For example:

```ini
privacy_policy_urls.en=https://example.com/en/privacy
privacy_policy_urls.el=https://example.com/el/privacy
```

Values must be nonempty; omitted entries keep the `/privacy` default.

The policy source in [privacy_policy.md](/home/dds/src/pixelbonus2/privacy_policy.md) still contains deployment-time placeholders:

- `[Insert Date]`
- `[Your Application Name]`
- `[Legal Entity Name]`
- `[Address]`
- `[Contact Email]`
