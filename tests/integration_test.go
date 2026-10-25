package tests

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"pixelbonus/src/courses"
)

func TestInstructorAndStudentFlow(t *testing.T) {
	ts, db, _ := newTestServer(t)
	defer ts.Close()

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	body := url.Values{"full_name": {"Teacher"}, "email": {"teacher@example.com"}, "password": {"password123"}, "csrf_token": {"YW5vbnltb3VzOjEwMDA6d3Jvbmc"}}.Encode()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/instructor/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected csrf protection, got %d", resp.StatusCode)
	}

	instructorID := upsertTestInstructor(t, db)
	courseSvc := &courses.Service{DB: db, Now: func() time.Time { return time.Unix(1000, 0) }}
	course, err := courseSvc.CreateCourse(testContext(), instructorID, courses.CreateCourseInput{Name: "SE 2026", GradingMethod: courses.ValidGradingMethods[0]})
	if err != nil {
		t.Fatal(err)
	}
	seedCoupon(t, db, course.ID, "00112233445566778899AABBCCDDEEFF00112233")

	for _, prefix := range []string{"/public/", "/course_overview/"} {
		resp, err = client.Get(ts.URL + prefix + course.PublicNonce)
		if err != nil {
			t.Fatal(err)
		}
		body, readErr := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil || resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "<h2>SE 2026</h2>") {
			t.Fatalf("%s page failed: %v %d", prefix, readErr, resp.StatusCode)
		}
		resp, err = client.Get(ts.URL + prefix + "unknown-course")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("%s unknown course: got %d", prefix, resp.StatusCode)
		}
	}
}
