package tests

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"pixelbonus/src/pixelbonus/auth"
	"pixelbonus/src/pixelbonus/coupon"
	"pixelbonus/src/pixelbonus/course"
	"pixelbonus/src/pixelbonus/email"
	"pixelbonus/src/pixelbonus/grade"
	"pixelbonus/src/pixelbonus/i18n"
	"pixelbonus/src/pixelbonus/web"
)

func TestWebCourseLifecycleAndRedeem(t *testing.T) {
	cfg := testConfig(t)
	db := testDB(t, cfg)
	defer db.Close()
	mailOut := filepath.Join(t.TempDir(), "mail.txt")
	authSvc := auth.NewService(db, email.NewSender(writeMailScript(t, mailOut), "no-reply@pixelbonus.com"), cfg.BaseURL, fixedTime, randomReader('a'))
	courseSvc := course.NewService(db, fixedTime)
	couponSvc := coupon.NewService(db, fixedTime, randomReader('b'))
	gradeSvc := grade.NewService(db)
	handler := web.NewServer(cfg, i18n.NewCatalog(), authSvc, courseSvc, couponSvc, gradeSvc)

	cookies := map[string]string{}
	body := perform(t, handler, cookies, http.MethodGet, "/instructor/register", nil)
	csrf := extractCSRF(body)
	body = perform(t, handler, cookies, http.MethodPost, "/instructor/register", url.Values{
		"csrf_token": {csrf},
		"full_name":  {"Alice"},
		"email":      {"alice@example.com"},
		"password":   {"password123"},
	})
	if !strings.Contains(body, "Registration completed") {
		t.Fatalf("unexpected body: %s", body)
	}

	var nonce string
	if err := db.QueryRow(`SELECT nonce FROM verification_tokens LIMIT 1`).Scan(&nonce); err != nil {
		t.Fatal(err)
	}
	perform(t, handler, cookies, http.MethodGet, "/instructor/verify?token="+nonce, nil)

	body = perform(t, handler, cookies, http.MethodGet, "/instructor/login", nil)
	csrf = extractCSRF(body)
	perform(t, handler, cookies, http.MethodPost, "/instructor/login", url.Values{
		"csrf_token": {csrf},
		"email":      {"alice@example.com"},
		"password":   {"password123"},
	})

	body = perform(t, handler, cookies, http.MethodGet, "/instructor/courses/new", nil)
	csrf = extractCSRF(body)
	perform(t, handler, cookies, http.MethodPost, "/instructor/courses/new", url.Values{
		"csrf_token":     {csrf},
		"name":           {"Course"},
		"grading_method": {"zero_to_q2_10"},
	})

	if _, err := db.Exec(`INSERT INTO coupon_generation_jobs (id, instructor_id, course_id, quantity, status, created_at) VALUES (1, 1, 1, 20, 'completed', ?)`, fixedTime()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO coupon_sets (id, course_id, job_id, created_at) VALUES (1, 1, 1, ?)`, fixedTime()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO coupons (coupon_set_id, course_id, raw_code, normalized_code) VALUES (1, 1, X'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA', 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA')`); err != nil {
		t.Fatal(err)
	}

	body = perform(t, handler, cookies, http.MethodGet, "/redeem", nil)
	csrf = cookies["pixelbonus_anon_csrf"]
	body = perform(t, handler, cookies, http.MethodPost, "/redeem", url.Values{
		"csrf_token": {csrf},
		"code":       {"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"},
		"student_id": {"s1"},
	})
	if !strings.Contains(body, "successfully redeemed") {
		t.Fatalf("unexpected redemption response: %s", body)
	}
}

func perform(t *testing.T, handler http.Handler, cookies map[string]string, method string, path string, form url.Values) string {
	t.Helper()
	var bodyReader *strings.Reader
	if form != nil {
		bodyReader = strings.NewReader(form.Encode())
	} else {
		bodyReader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for k, v := range cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	for _, c := range rec.Result().Cookies() {
		cookies[c.Name] = c.Value
	}
	data, err := ioutil.ReadAll(rec.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func extractCSRF(body string) string {
	start := strings.Index(body, `name="csrf_token" value="`)
	if start == -1 {
		return ""
	}
	start += len(`name="csrf_token" value="`)
	end := strings.Index(body[start:], `"`)
	return body[start : start+end]
}
