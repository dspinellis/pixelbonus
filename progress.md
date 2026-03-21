app | integrated | src/main.go | tests/app_test.go
config | integrated | src/pixelbonus/config/config.go | tests/config_test.go
database | integrated | src/pixelbonus/database/database.go third_party/github.com/mattn/go-sqlite3/sqlite3.go | tests/database_test.go
schema | integrated | design/schema.sql | tests/database_test.go
i18n | integrated | src/pixelbonus/i18n/i18n.go | tests/i18n_test.go
auth | integrated | src/pixelbonus/auth/service.go third_party/golang.org/x/crypto/bcrypt/bcrypt.go | tests/auth_service_test.go
course | integrated | src/pixelbonus/course/service.go | tests/course_service_test.go
coupon | integrated | src/pixelbonus/coupon/service.go | tests/coupon_service_test.go
grade | integrated | src/pixelbonus/grade/service.go | tests/grade_service_test.go
job | integrated | src/pixelbonus/job/worker.go | tests/job_worker_test.go
pdf | integrated | src/pixelbonus/pdf/render.go | tests/pdf_render_test.go
email | integrated | src/pixelbonus/email/sender.go | tests/email_sender_test.go
web | integrated | src/pixelbonus/web/handlers.go | tests/web_integration_test.go
templates | integrated | src/templates | tests/web_integration_test.go
assets | integrated | src/assets | tests/web_integration_test.go
service-unit | integrated | src/pixelbonus.service | tests/readme_smoke_test.go
docs | integrated | README.md implementation-notes.md | tests/readme_smoke_test.go
