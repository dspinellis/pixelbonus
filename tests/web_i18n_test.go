package tests

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"pixelbonus/src/courses"
	"pixelbonus/src/i18n"
	"pixelbonus/src/redemption"
	"pixelbonus/src/session"
	"pixelbonus/src/web"
)

func TestLocalizedPrivacyPolicyLinks(t *testing.T) {
	catalogs, err := i18n.LoadCatalogs(filepath.Join("..", "src", "translations"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name       string
		urls       map[string]string
		lang, want string
	}{
		{"English", map[string]string{"en": "https://example.com/en/privacy", "el": "https://example.com/el/privacy"}, "en", "https://example.com/en/privacy"},
		{"Greek", map[string]string{"en": "https://example.com/en/privacy", "el": "https://example.com/el/privacy"}, "el", "https://example.com/el/privacy"},
		{"default English", nil, "en", "/privacy"},
		{"default Greek", map[string]string{"en": "/english-policy"}, "el", "/privacy"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, err := web.NewServer(web.Dependencies{
				TemplateDir:       filepath.Join("..", "src", "templates"),
				Catalogs:          catalogs,
				PrivacyPolicyURLs: tc.urls,
			})
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Accept-Language", tc.lang)
			response := httptest.NewRecorder()
			server.Routes().ServeHTTP(response, req)
			if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `href="`+tc.want+`"`) {
				t.Fatalf("missing policy link %q: %d %s", tc.want, response.Code, response.Body.String())
			}
		})
	}
}

func TestGreekPagesFollowBrowserPreferences(t *testing.T) {
	ts, db, _ := newTestServer(t)
	defer ts.Close()
	id := upsertTestInstructor(t, db)
	course, err := (&courses.Service{DB: db}).CreateCourse(testContext(), id, courses.CreateCourseInput{Name: "Δοκιμή", GradingMethod: courses.ValidGradingMethods[0]})
	if err != nil {
		t.Fatal(err)
	}
	sess, err := (&session.Store{DB: db}).Create(testContext(), id, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	code := "00112233445566778899AABBCCDDEEFF00112233"
	seedCoupon(t, db, course.ID, code)
	coursePath := "/instructor/courses/" + strconv.FormatInt(course.ID, 10)
	for _, tc := range []struct {
		path  string
		auth  bool
		words []string
	}{
		{"/", false, []string{"Η συμμετοχή σας μετράει!", "Αγγλικά", "Ελληνικά"}},
		{"/public/" + course.PublicNonce, false, []string{"<th>Αριθμός μητρώου</th>", "<th>Εξαργυρώσεις</th>"}},
		{"/course_overview/" + course.PublicNonce, false, []string{"<th>Αριθμός μητρώου</th>", "<th>Εξαργυρώσεις</th>"}},
		{coursePath + "/grades", true, []string{"<th>Αριθμός μητρώου</th>", "<th>Εξαργυρώσεις</th>", "<th>Βαθμός</th>", "<th>Κατάταξη</th>"}},
		{"/instructor/courses", true, []string{"Νέο μάθημα", `title="Παραγωγή κουπονιών QR"`, `title="Ρυθμίσεις"`}},
		{"/instructor/courses/new", true, []string{"Όνομα μαθήματος", "Μέθοδος βαθμολόγησης", "Αποθήκευση"}},
		{coursePath + "/settings", true, []string{"Όνομα μαθήματος", "Μέθοδος βαθμολόγησης", "Αποθήκευση"}},
		{coursePath + "/tokens/new", true, []string{"Πλήθος κουπονιών", "Δημιουργία"}},
		{"/instructor/profile", true, []string{"Τρέχων κωδικός", "Νέος κωδικός", "Αποθήκευση"}},
		{"/instructor/login", false, []string{"Κωδικός", "Ξέχασα τον κωδικό μου", "Εγγραφή", "Υποβολή"}},
		{"/instructor/register", false, []string{"Ονοματεπώνυμο", "Κωδικός", "Υποβολή"}},
		{"/instructor/password/forgot", false, []string{"Email", "Υποβολή"}},
		{"/instructor/password/reset?token=example", false, []string{"Νέος κωδικός", "Υποβολή"}},
		{"/redeem?hash=" + code, false, []string{"Κωδικός κουπονιού", "Αριθμός μητρώου", "Εξαργύρωση κουπονιού"}},
		{"/r?c=" + redemption.EncodeTokenForURL(code), false, []string{"Κωδικός κουπονιού", "Αριθμός μητρώου", "Εξαργύρωση κουπονιού"}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			req, _ := http.NewRequest("GET", ts.URL+tc.path, nil)
			req.Header.Set("Accept-Language", "fr-FR,el-GR;q=0.9,en;q=0.3")
			if tc.auth {
				req.AddCookie(&http.Cookie{Name: "pixelbonus_session", Value: sess.ID})
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			data, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Language") != "el" || !strings.Contains(string(data), `<html lang="el">`) {
				t.Fatalf("Greek page not selected: %d %s", resp.StatusCode, data)
			}
			for _, word := range tc.words {
				if !strings.Contains(string(data), word) {
					t.Fatalf("missing Greek text %q", word)
				}
			}
		})
	}
	// An explicit saved choice takes priority over browser preferences.
	req, _ := http.NewRequest("GET", ts.URL+"/public/"+course.PublicNonce, nil)
	req.Header.Set("Accept-Language", "el-GR")
	req.AddCookie(&http.Cookie{Name: "pixelbonus_lang", Value: "en"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.Header.Get("Content-Language") != "en" || !strings.Contains(string(data), "<th>Student identifier</th>") || !strings.Contains(string(data), "<th>Redemptions</th>") {
		t.Fatalf("saved English selection not respected: %s", data)
	}
}
