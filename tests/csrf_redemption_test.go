package tests

import (
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"pixelbonus/src/courses"
	"pixelbonus/src/redemption"
)

func TestFirstVisitCouponRedemptionCSRF(t *testing.T) {
	for _, mode := range []string{"manual", "qr"} {
		t.Run(mode, func(t *testing.T) {
			original, db, _ := newTestServer(t)
			defer original.Close()
			ts := httptest.NewTLSServer(original.Config.Handler)
			defer ts.Close()
			id := upsertTestInstructor(t, db)
			course, err := (&courses.Service{DB: db}).CreateCourse(testContext(), id, courses.CreateCourseInput{Name: "CSRF", GradingMethod: courses.ValidGradingMethods[0]})
			if err != nil {
				t.Fatal(err)
			}
			code := "00112233445566778899AABBCCDDEEFF00112233"
			seedCoupon(t, db, course.ID, code)
			path := "/redeem?hash=" + code
			if mode == "qr" {
				path = "/r?c=" + redemption.EncodeTokenForURL(code)
			}
			client := ts.Client()
			client.Jar, _ = cookiejar.New(nil)
			resp, err := client.Get(ts.URL + path)
			if err != nil {
				t.Fatal(err)
			}
			data, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			token := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`).FindStringSubmatch(string(data))
			if len(token) != 2 {
				t.Fatal("missing CSRF form token")
			}
			cookies := resp.Cookies()
			var visitor string
			for _, c := range cookies {
				if c.Name == "pixelbonus_vid" {
					visitor = c.Value
					if !c.Secure || !c.HttpOnly {
						t.Fatal("visitor cookie lacks security attributes")
					}
				}
			}
			if visitor == "" {
				t.Fatal("missing first-visit cookie")
			}
			other := &http.Client{Transport: ts.Client().Transport}
			other.Jar, _ = cookiejar.New(nil)
			resp, err = other.Get(ts.URL + path)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			for _, c := range resp.Cookies() {
				if c.Name == "pixelbonus_vid" && c.Value == visitor {
					t.Fatal("visitors share an ID")
				}
			}
			values := url.Values{"csrf_token": {token[1]}, "code": {code}, "student_id": {"student-1"}}
			resp, err = other.PostForm(ts.URL+path, values)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("another visitor's token accepted: %d", resp.StatusCode)
			}
			resp, err = client.PostForm(ts.URL+path, values)
			if err != nil {
				t.Fatal(err)
			}
			data, _ = ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK || !strings.Contains(string(data), "successfully redeemed") {
				t.Fatalf("first-visit redemption failed: %d %s", resp.StatusCode, data)
			}
			var count int
			if err := db.QueryRow("SELECT COUNT(*) FROM Redemption WHERE qrcode_code = ? AND participantNumber = ?", code, "student-1").Scan(&count); err != nil || count != 1 {
				t.Fatalf("redemption not recorded: %d %v", count, err)
			}
		})
	}
}
