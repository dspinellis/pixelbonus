package tests

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"pixelbonus/src/courses"
	"pixelbonus/src/redemption"
	"pixelbonus/src/session"
)

func TestSignedInHomeNavigation(t *testing.T) {
	ts, db, _ := newTestServer(t)
	defer ts.Close()
	id := upsertTestInstructor(t, db)
	store := &session.Store{DB: db}
	sess, err := store.Create(testContext(), id, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for _, path := range []string{"/", "/instructor/login", "/instructor/courses", "/"} {
		req, _ := http.NewRequest("GET", ts.URL+path, nil)
		req.AddCookie(&http.Cookie{Name: "pixelbonus_session", Value: sess.ID})
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if path == "/instructor/login" {
			if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/instructor/courses" {
				t.Fatalf("signed-in login: %d %s", resp.StatusCode, resp.Header.Get("Location"))
			}
		} else if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: %d", path, resp.StatusCode)
		}
		if path == "/" && (!strings.Contains(string(data), `href="/instructor/courses"`) || strings.Contains(string(data), `href="/instructor/login"`)) {
			t.Fatal("home should link to dashboard for signed-in instructor")
		}
	}
}

func TestLanguageSwitchKeepsCurrentURL(t *testing.T) {
	ts, _, _ := newTestServer(t)
	defer ts.Close()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for _, target := range []string{"/redeem?code=ABC", "https://example.com/", "//example.com/"} {
		resp, err := client.PostForm(ts.URL+"/lang", url.Values{"lang": {"el"}, "return_to": {target}})
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		want := target
		if target != "/redeem?code=ABC" {
			want = "/"
		}
		if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != want {
			t.Fatalf("language redirect: %d %s", resp.StatusCode, resp.Header.Get("Location"))
		}
		found := false
		for _, c := range resp.Cookies() {
			if c.Name == "pixelbonus_lang" && c.Value == "el" {
				found = true
			}
		}
		if !found {
			t.Fatal("missing language cookie")
		}
	}
}

func TestRedemptionURLPrefillsCode(t *testing.T) {
	ts, db, _ := newTestServer(t)
	defer ts.Close()
	id := upsertTestInstructor(t, db)
	course, err := (&courses.Service{DB: db}).CreateCourse(testContext(), id, courses.CreateCourseInput{Name: "Testing", GradingMethod: courses.ValidGradingMethods[0]})
	if err != nil {
		t.Fatal(err)
	}
	code := "00112233445566778899AABBCCDDEEFF00112233"
	seedCoupon(t, db, course.ID, code)
	encoded := redemption.EncodeTokenForURL(code)
	for _, path := range []string{"/redeem?hash=" + code, "/redeem?code=" + code, "/redeem?c=" + code, "/redeem?c=" + encoded, "/r?c=" + encoded} {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || !strings.Contains(string(data), `value="`+code+`"`) {
			t.Fatalf("coupon missing from %s: %d", path, resp.StatusCode)
		}
	}
}
