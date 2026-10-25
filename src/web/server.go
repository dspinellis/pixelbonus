package web

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"html"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"pixelbonus/src/auth"
	"pixelbonus/src/courses"
	"pixelbonus/src/csrf"
	"pixelbonus/src/grades"
	"pixelbonus/src/i18n"
	"pixelbonus/src/public"
	"pixelbonus/src/redemption"
	"pixelbonus/src/session"
	"pixelbonus/src/tokens"
)

type Dependencies struct {
	PrivacyPolicyURLs map[string]string
	TemplateDir       string
	Catalogs          i18n.Catalogs
	Auth              *auth.Service
	Sessions          *session.Store
	CSRF              *csrf.Service
	Courses           *courses.Service
	Tokens            *tokens.Service
	Grades            *grades.Service
	Public            *public.Service
	Redemption        *redemption.Service
	DB                *sql.DB
	Logger            interface {
		Log(string, string, map[string]string)
	}
}

type Server struct {
	deps      Dependencies
	templates map[string]*template.Template
}

func NewServer(deps Dependencies) (*Server, error) {
	s := &Server{deps: deps, templates: map[string]*template.Template{}}
	for _, lang := range []string{"en", "el"} {
		t, err := template.ParseFiles(filepath.Join(deps.TemplateDir, lang, "page.html"))
		if err != nil {
			return nil, err
		}
		s.templates[lang] = t
	}
	return s, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.home)
	mux.HandleFunc("/lang", s.lang)
	mux.HandleFunc("/privacy", s.privacy)
	mux.HandleFunc("/redeem", s.redeemManual)
	mux.HandleFunc("/r", s.redeemQR)
	mux.HandleFunc("/instructor/login", s.login)
	mux.HandleFunc("/instructor/register", s.register)
	mux.HandleFunc("/instructor/register/confirm", s.confirm)
	mux.HandleFunc("/instructor/password/forgot", s.forgot)
	mux.HandleFunc("/instructor/password/reset", s.reset)
	mux.HandleFunc("/instructor/profile", s.profile)
	mux.HandleFunc("/instructor/logout", s.logout)
	mux.HandleFunc("/instructor/courses", s.coursesIndex)
	mux.HandleFunc("/instructor/courses/new", s.courseNew)
	mux.HandleFunc("/instructor/courses/", s.courseSubroutes)
	mux.HandleFunc("/public/", s.publicPage)
	mux.HandleFunc("/course_overview/", s.publicPage)
	mux.HandleFunc("/download/", s.download)
	return s.middleware(mux)
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.deps.Logger != nil {
			started := time.Now()
			response := &loggedResponse{ResponseWriter: w, status: http.StatusOK}
			w = response
			defer func() {
				s.deps.Logger.Log("INFO", "http_request", map[string]string{
					"method":   r.Method,
					"path":     r.URL.Path,
					"status":   strconv.Itoa(response.status),
					"duration": time.Since(started).String(),
				})
			}()
		}
		if _, err := r.Cookie("pixelbonus_vid"); err != nil {
			raw := make([]byte, 18)
			if _, err := rand.Read(raw); err != nil {
				http.Error(w, s.deps.Catalogs.T(s.language(r), "browser_session_failed", nil), http.StatusInternalServerError)
				return
			}
			cookie := &http.Cookie{Name: "pixelbonus_vid", Value: base64.RawURLEncoding.EncodeToString(raw), Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: true}
			http.SetCookie(w, cookie)
			// Bind the first rendered form to the same visitor ID that the
			// browser will send when submitting it.
			r.AddCookie(cookie)
		}
		next.ServeHTTP(w, r)
	})
}

type loggedResponse struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *loggedResponse) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *loggedResponse) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, title string, content template.HTML, status int) {
	lang := s.language(r)
	privacyPolicyURL := s.deps.PrivacyPolicyURLs[lang]
	if privacyPolicyURL == "" {
		privacyPolicyURL = "/privacy"
	}
	data := map[string]interface{}{
		"PrivacyPolicyURL": privacyPolicyURL,
		"Title":            title,
		"Content":          content,
		"Lang":             lang,
		"ReturnTo":         r.URL.RequestURI(),
		"T":                func(key string) string { return s.deps.Catalogs.T(lang, key, nil) },
		"CurrentUser":      s.currentUserName(r),
		"Icons": map[string]template.HTML{
			"home":     iconHome(),
			"user":     iconUser(),
			"settings": iconSettings(),
			"github":   iconGitHub(),
		},
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Language", lang)
	w.Header().Add("Vary", "Accept-Language")
	w.Header().Add("Vary", "Cookie")
	w.WriteHeader(status)
	_ = s.templates[lang].Execute(w, data)
}

func (s *Server) text(r *http.Request, key string) string {
	return html.EscapeString(s.deps.Catalogs.T(s.language(r), key, nil))
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, "PixelBonus", template.HTML("<p>"+s.text(r, "not_found")+"</p>"), http.StatusNotFound)
}

func (s *Server) language(r *http.Request) string {
	explicit := r.FormValue("lang")
	cookie := ""
	if c, err := r.Cookie("pixelbonus_lang"); err == nil {
		cookie = c.Value
	}
	return i18n.ResolveLanguage(explicit, cookie, r.Header.Get("Accept-Language"))
}

func (s *Server) visitorID(r *http.Request) string {
	if c, err := r.Cookie("pixelbonus_vid"); err == nil {
		return c.Value
	}
	return "anonymous"
}

func (s *Server) csrfToken(r *http.Request) string {
	token, _ := s.deps.CSRF.Issue(s.visitorID(r))
	return token
}

func (s *Server) requireCSRF(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return true
	}
	if !s.deps.CSRF.Verify(s.visitorID(r), r.FormValue("csrf_token")) {
		s.render(w, r, "PixelBonus", template.HTML("<p>"+s.text(r, "invalid_csrf")+"</p>"), http.StatusForbidden)
		return false
	}
	return true
}

func (s *Server) currentUser(r *http.Request) (session.SessionData, bool) {
	c, err := r.Cookie("pixelbonus_session")
	if err != nil {
		return session.SessionData{}, false
	}
	data, err := s.deps.Sessions.Validate(r.Context(), c.Value, time.Now())
	return data, err == nil
}

func (s *Server) currentUserName(r *http.Request) string {
	user, ok := s.currentUser(r)
	if !ok {
		return ""
	}
	var name string
	_ = s.deps.DB.QueryRowContext(r.Context(), "SELECT COALESCE(name, username) FROM Users WHERE id = ?", user.InstructorID).Scan(&name)
	return name
}

func (s *Server) mustAuth(w http.ResponseWriter, r *http.Request) (session.SessionData, bool) {
	user, ok := s.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/instructor/login", http.StatusSeeOther)
		return session.SessionData{}, false
	}
	return user, true
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	lang := s.language(r)
	t := func(key string) string { return s.deps.Catalogs.T(lang, key, nil) }
	instructorURL, instructorLabel := "/instructor/login", t("go_login")
	if _, ok := s.currentUser(r); ok {
		instructorURL, instructorLabel = "/instructor/courses", t("instructor_dashboard")
	}
	body := `<h1>` + html.EscapeString(t("home_heading")) + `</h1>` +
		`<section class="split"><div class="panel">` + string(iconUser()) + `<h2>` + html.EscapeString(t("participant_title")) + `</h2><p>` + html.EscapeString(t("participant_body")) + `</p><p><a class="button" href="/redeem">` + html.EscapeString(t("go_redeem")) + `</a></p></div>` +
		`<div class="panel">` + string(iconAcademicCap()) + `<h2>` + html.EscapeString(t("instructor_title")) + `</h2><p>` + html.EscapeString(t("instructor_body")) + `</p><p><a class="button" href="` + instructorURL + `">` + html.EscapeString(instructorLabel) + `</a></p></div></section>`
	s.render(w, r, "PixelBonus", template.HTML(body), http.StatusOK)
}

func (s *Server) lang(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		http.SetCookie(w, &http.Cookie{Name: "pixelbonus_lang", Value: s.language(r), Path: "/", HttpOnly: false, SameSite: http.SameSiteLaxMode, Secure: true})
		target := r.FormValue("return_to")
		u, err := url.Parse(target)
		if err != nil || !strings.HasPrefix(target, "/") || strings.HasPrefix(target, "//") || u.Host != "" || u.Scheme != "" {
			target = "/"
		}
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) privacy(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadFile("privacy_policy.md")
	if err != nil {
		s.render(w, r, "PixelBonus", template.HTML("<p>"+s.text(r, "privacy_missing")+"</p>"), http.StatusInternalServerError)
		return
	}
	s.render(w, r, "PixelBonus", markdownToHTML(string(b)), http.StatusOK)
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if !s.requireCSRF(w, r) {
		return
	}
	if r.Method == http.MethodPost {
		_, err := s.deps.Auth.RegisterInstructor(r.Context(), auth.RegisterInput{FullName: r.FormValue("full_name"), Email: r.FormValue("email"), Password: r.FormValue("password")})
		if err == nil {
			s.render(w, r, "PixelBonus", template.HTML("<p>"+html.EscapeString(s.deps.Catalogs.T(s.language(r), "verification_sent", nil))+"</p>"), http.StatusOK)
			return
		}
	}
	content := s.formPage(r, "/instructor/register", s.csrfToken(r), []field{{"full_name", "text"}, {"email", "email"}, {"password", "password"}})
	s.render(w, r, "PixelBonus", content, http.StatusOK)
}

func (s *Server) confirm(w http.ResponseWriter, r *http.Request) {
	if _, err := s.deps.Auth.ConfirmInstructorEmail(r.Context(), r.URL.Query().Get("token")); err != nil {
		s.render(w, r, "PixelBonus", template.HTML("<p>"+s.text(r, "invalid_verification")+"</p>"), http.StatusBadRequest)
		return
	}
	s.render(w, r, "PixelBonus", template.HTML("<p>"+html.EscapeString(s.deps.Catalogs.T(s.language(r), "verification_done", nil))+` <a href="/instructor/login">`+s.text(r, "login")+`</a></p>`), http.StatusOK)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(r); ok {
		http.Redirect(w, r, "/instructor/courses", http.StatusSeeOther)
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	if r.Method == http.MethodPost {
		ident, err := s.deps.Auth.Login(r.Context(), r.FormValue("email"), r.FormValue("password"))
		if err == nil {
			sess, err := s.deps.Sessions.Create(r.Context(), ident.ID, time.Now())
			if err == nil {
				http.SetCookie(w, &http.Cookie{Name: "pixelbonus_session", Value: sess.ID, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: true})
				http.Redirect(w, r, "/instructor/courses", http.StatusSeeOther)
				return
			}
		}
	}
	body := s.formPage(r, "/instructor/login", s.csrfToken(r), []field{{"email", "email"}, {"password", "password"}}) + template.HTML(`<p><a href="/instructor/password/forgot">`+s.text(r, "forgot_password")+`</a></p><p><a href="/instructor/register">`+s.text(r, "register")+`</a></p>`)
	s.render(w, r, "PixelBonus", body, http.StatusOK)
}

func (s *Server) forgot(w http.ResponseWriter, r *http.Request) {
	if !s.requireCSRF(w, r) {
		return
	}
	if r.Method == http.MethodPost {
		_, _ = s.deps.Auth.CreatePasswordReset(r.Context(), r.FormValue("email"))
		s.render(w, r, "PixelBonus", template.HTML("<p>"+html.EscapeString(s.deps.Catalogs.T(s.language(r), "ack_reset", nil))+"</p>"), http.StatusOK)
		return
	}
	s.render(w, r, "PixelBonus", s.formPage(r, "/instructor/password/forgot", s.csrfToken(r), []field{{"email", "email"}}), http.StatusOK)
}

func (s *Server) reset(w http.ResponseWriter, r *http.Request) {
	if !s.requireCSRF(w, r) {
		return
	}
	if r.Method == http.MethodPost {
		if _, err := s.deps.Auth.ResetPassword(r.Context(), r.FormValue("token"), r.FormValue("password")); err == nil {
			http.Redirect(w, r, "/instructor/login", http.StatusSeeOther)
			return
		}
	}
	body := `<form method="post"><input type="hidden" name="csrf_token" value="` + html.EscapeString(s.csrfToken(r)) + `"><input type="hidden" name="token" value="` + html.EscapeString(r.URL.Query().Get("token")) + `"><label>` + s.text(r, "new_password") + `<input type="password" name="password"></label><button type="submit">` + s.text(r, "submit") + `</button></form>`
	s.render(w, r, "PixelBonus", template.HTML(body), http.StatusOK)
}

func (s *Server) profile(w http.ResponseWriter, r *http.Request) {
	user, ok := s.mustAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	if r.Method == http.MethodPost {
		if err := s.deps.Auth.ChangePassword(r.Context(), user.InstructorID, r.FormValue("current_password"), r.FormValue("password")); err == nil {
			http.Redirect(w, r, "/instructor/courses", http.StatusSeeOther)
			return
		}
	}
	body := `<form method="post"><input type="hidden" name="csrf_token" value="` + html.EscapeString(s.csrfToken(r)) + `"><label>` + s.text(r, "current_password") + `<input type="password" name="current_password"></label><label>` + s.text(r, "new_password") + `<input type="password" name="password"></label><button type="submit">` + s.text(r, "save") + `</button></form>`
	s.render(w, r, "PixelBonus", template.HTML(body), http.StatusOK)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("pixelbonus_session"); err == nil {
		_ = s.deps.Sessions.Destroy(r.Context(), c.Value)
		http.SetCookie(w, &http.Cookie{Name: "pixelbonus_session", Value: "", Path: "/", MaxAge: -1})
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) coursesIndex(w http.ResponseWriter, r *http.Request) {
	user, ok := s.mustAuth(w, r)
	if !ok {
		return
	}
	list, err := s.deps.Courses.ListCourses(r.Context(), user.InstructorID)
	if err != nil {
		s.render(w, r, "PixelBonus", template.HTML("<p>"+s.text(r, "courses_load_failed")+"</p>"), http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString(`<p><a class="button" href="/instructor/courses/new">` + s.text(r, "new_course") + `</a></p><ul class="courses">`)
	for _, c := range list {
		id := strconv.FormatInt(c.ID, 10)
		b.WriteString(`<li><strong>` + html.EscapeString(c.Name) + `</strong><span class="course-actions">` +
			`<a title="` + s.text(r, "generate_tokens") + `" aria-label="` + s.text(r, "generate_tokens") + `" href="/instructor/courses/` + id + `/tokens/new">` + string(iconNew()) + `</a> ` +
			`<a title="` + s.text(r, "grades") + `" aria-label="` + s.text(r, "grades") + `" href="/instructor/courses/` + id + `/grades">` + string(iconGrades()) + `</a> ` +
			`<a title="` + s.text(r, "download_csv") + `" href="/instructor/courses/` + id + `/grades.csv">CSV</a> ` +
			`<a title="` + s.text(r, "public_page") + `" aria-label="` + s.text(r, "public_page") + `" href="/public/` + html.EscapeString(c.PublicNonce) + `">` + string(iconPublic()) + `</a> ` +
			`<a title="` + s.text(r, "settings") + `" aria-label="` + s.text(r, "settings") + `" href="/instructor/courses/` + id + `/settings">` + string(iconSettings()) + `</a></span></li>`)
	}
	b.WriteString(`</ul>`)
	s.render(w, r, "PixelBonus", template.HTML(b.String()), http.StatusOK)
}

func (s *Server) courseNew(w http.ResponseWriter, r *http.Request) {
	user, ok := s.mustAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	if r.Method == http.MethodPost {
		_, err := s.deps.Courses.CreateCourse(r.Context(), user.InstructorID, courses.CreateCourseInput{Name: r.FormValue("name"), GradingMethod: r.FormValue("grading_method")})
		if err == nil {
			http.Redirect(w, r, "/instructor/courses", http.StatusSeeOther)
			return
		}
	}
	s.render(w, r, "PixelBonus", s.courseForm(r, "/instructor/courses/new", s.csrfToken(r), "", ""), http.StatusOK)
}

func (s *Server) courseSubroutes(w http.ResponseWriter, r *http.Request) {
	user, ok := s.mustAuth(w, r)
	if !ok {
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/instructor/courses/"), "/")
	if len(parts) < 2 {
		s.notFound(w, r)
		return
	}
	courseID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		s.notFound(w, r)
		return
	}
	switch parts[1] {
	case "grades.csv":
		cg, err := s.deps.Grades.ComputeCourseGrades(r.Context(), user.InstructorID, courseID)
		if err != nil {
			s.notFound(w, r)
			return
		}
		data, _ := grades.RenderCSV(cg.Rows)
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write(data)
		return
	case "settings":
		if !s.requireCSRF(w, r) {
			return
		}
		course, err := s.deps.Courses.GetCourse(r.Context(), user.InstructorID, courseID)
		if err != nil {
			s.notFound(w, r)
			return
		}
		if r.Method == http.MethodPost {
			_, err := s.deps.Courses.UpdateCourse(r.Context(), user.InstructorID, courses.UpdateCourseInput{ID: courseID, Name: r.FormValue("name"), GradingMethod: r.FormValue("grading_method")})
			if err == nil {
				http.Redirect(w, r, "/instructor/courses", http.StatusSeeOther)
				return
			}
		}
		s.render(w, r, "PixelBonus", s.courseForm(r, r.URL.Path, s.csrfToken(r), course.Name, course.GradingMethod), http.StatusOK)
	case "tokens":
		if len(parts) == 3 && parts[2] == "new" {
			if !s.requireCSRF(w, r) {
				return
			}
			if r.Method == http.MethodPost {
				count, _ := strconv.Atoi(r.FormValue("count"))
				if _, err := s.deps.Tokens.CreateTokenSet(r.Context(), user.InstructorID, courseID, count, time.Now()); err == nil {
					s.render(w, r, "PixelBonus", template.HTML("<p>"+html.EscapeString(s.deps.Catalogs.T(s.language(r), "ack_tokens", nil))+"</p>"), http.StatusOK)
					return
				}
			}
			body := `<form method="post"><input type="hidden" name="csrf_token" value="` + html.EscapeString(s.csrfToken(r)) + `"><label>` + s.text(r, "token_count") + `<input type="number" min="20" step="20" name="count"></label><button type="submit">` + s.text(r, "create") + `</button></form>`
			s.render(w, r, "PixelBonus", template.HTML(body), http.StatusOK)
			return
		}
		s.notFound(w, r)
	case "grades":
		cg, err := s.deps.Grades.ComputeCourseGrades(r.Context(), user.InstructorID, courseID)
		if err != nil {
			s.notFound(w, r)
			return
		}
		var b strings.Builder
		b.WriteString(`<table><tr><th>` + s.text(r, "student_id") + `</th><th>` + s.text(r, "redemptions") + `</th><th>` + s.text(r, "grade") + `</th><th>` + s.text(r, "rank") + `</th></tr>`)
		for _, row := range cg.Rows {
			b.WriteString(`<tr><td>` + html.EscapeString(row.StudentID) + `</td><td>` + strconv.Itoa(row.Redemptions) + `</td><td>` + strconv.Itoa(row.Grade) + `</td><td>` + strconv.Itoa(row.Rank) + `</td></tr>`)
		}
		b.WriteString(`</table>`)
		s.render(w, r, "PixelBonus", template.HTML(b.String()), http.StatusOK)
	default:
		s.notFound(w, r)
	}
}

func (s *Server) publicPage(w http.ResponseWriter, r *http.Request) {
	nonce := strings.TrimPrefix(r.URL.Path, "/public/")
	if strings.HasPrefix(r.URL.Path, "/course_overview/") {
		nonce = strings.TrimPrefix(r.URL.Path, "/course_overview/")
	}
	view, err := s.deps.Public.ListPublicRedemptions(r.Context(), nonce)
	if err != nil {
		s.notFound(w, r)
		return
	}
	var b strings.Builder
	b.WriteString(`<h2>` + html.EscapeString(view.Course.Name) + `</h2><table><tr><th>` + s.text(r, "student_id") + `</th><th>` + s.text(r, "redemptions") + `</th></tr>`)
	for _, row := range view.Rows {
		b.WriteString(`<tr><td>` + html.EscapeString(row.StudentID) + `</td><td>` + strconv.Itoa(row.Redemptions) + `</td></tr>`)
	}
	b.WriteString(`</table>`)
	s.render(w, r, "PixelBonus", template.HTML(b.String()), http.StatusOK)
}

func (s *Server) redeemManual(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	if r.Method == http.MethodGet && code == "" {
		code = r.URL.Query().Get("hash")
	}
	if r.Method == http.MethodGet && code == "" && r.URL.Query().Get("c") != "" {
		page, err := s.deps.Redemption.LookupRedeemPage(r.Context(), r.URL.Query().Get("c"))
		if err == nil {
			code = page.Code
		} else {
			code = r.URL.Query().Get("c")
		}
	}
	if !s.requireCSRF(w, r) {
		return
	}
	if r.Method == http.MethodPost {
		res, err := s.deps.Redemption.Redeem(r.Context(), redemption.RedeemInput{Code: r.FormValue("code"), StudentID: r.FormValue("student_id")}, time.Now())
		if err == nil {
			s.render(w, r, "PixelBonus", template.HTML("<p>"+html.EscapeString(s.messageFor(r, res))+"</p>"), http.StatusOK)
			return
		}
	}
	body := `<form method="post"><input type="hidden" name="csrf_token" value="` + html.EscapeString(s.csrfToken(r)) + `"><label>` + s.text(r, "coupon_code") + `<input name="code" pattern="[0-9A-Fa-f ]+" value="` + html.EscapeString(code) + `"></label><label>` + s.text(r, "student_id") + `<input name="student_id" maxlength="20"></label><button type="submit">` + s.text(r, "redeem") + `</button></form>`
	s.render(w, r, "PixelBonus", template.HTML(body), http.StatusOK)
}

func (s *Server) redeemQR(w http.ResponseWriter, r *http.Request) {
	page, err := s.deps.Redemption.LookupRedeemPage(r.Context(), r.URL.Query().Get("c"))
	if err != nil {
		s.notFound(w, r)
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	if r.Method == http.MethodPost {
		res, err := s.deps.Redemption.Redeem(r.Context(), redemption.RedeemInput{Code: page.Code, StudentID: r.FormValue("student_id")}, time.Now())
		if err == nil {
			s.render(w, r, "PixelBonus", template.HTML("<p>"+html.EscapeString(s.messageFor(r, res))+"</p>"), http.StatusOK)
			return
		}
	}
	body := `<form method="post"><input type="hidden" name="csrf_token" value="` + html.EscapeString(s.csrfToken(r)) + `"><p>` + html.EscapeString(page.CourseName) + `</p><label>` + s.text(r, "coupon_code") + `<input name="code" value="` + html.EscapeString(page.Code) + `" readonly></label><label>` + s.text(r, "student_id") + `<input name="student_id" maxlength="20"></label><button type="submit">` + s.text(r, "redeem") + `</button></form>`
	s.render(w, r, "PixelBonus", template.HTML(body), http.StatusOK)
}

func (s *Server) messageFor(r *http.Request, res redemption.RedeemResult) string {
	lang := s.language(r)
	switch res.Status {
	case "success":
		return s.deps.Catalogs.T(lang, "success_redeemed", map[string]string{"code": res.Code, "course": res.CourseName, "student": res.StudentID})
	case "already":
		return s.deps.Catalogs.T(lang, "already_redeemed", map[string]string{"code": res.Code, "course": res.CourseName})
	default:
		return s.deps.Catalogs.T(lang, "invalid_coupon", map[string]string{"code": res.Code})
	}
}

func (s *Server) download(w http.ResponseWriter, r *http.Request) {
	path, course, err := s.deps.Tokens.GetDownloadPath(r.Context(), strings.TrimPrefix(r.URL.Path, "/download/"))
	if err != nil {
		s.notFound(w, r)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+sanitizeFilename(course)+`.pdf"`)
	http.ServeFile(w, r, path)
}

type field struct {
	Name string
	Type string
}

func (s *Server) formPage(r *http.Request, action string, csrfToken string, fields []field) template.HTML {
	var b strings.Builder
	b.WriteString(`<form method="post" action="` + action + `"><input type="hidden" name="csrf_token" value="` + html.EscapeString(csrfToken) + `">`)
	for _, f := range fields {
		b.WriteString(`<label>` + s.text(r, f.Name) + `<input type="` + html.EscapeString(f.Type) + `" name="` + html.EscapeString(f.Name) + `"></label>`)
	}
	b.WriteString(`<button type="submit">` + s.text(r, "submit") + `</button></form>`)
	return template.HTML(b.String())
}

func (s *Server) courseForm(r *http.Request, action string, csrfToken string, name string, selected string) template.HTML {
	var b strings.Builder
	b.WriteString(`<form method="post" action="` + action + `"><input type="hidden" name="csrf_token" value="` + html.EscapeString(csrfToken) + `"><label>` + s.text(r, "course_name") + `<input name="name" value="` + html.EscapeString(name) + `"></label><fieldset><legend>` + s.text(r, "grading_method") + `</legend>`)
	for _, method := range courses.ValidGradingMethods {
		checked := ""
		if method == selected {
			checked = ` checked`
		}
		b.WriteString(`<label class="grading-method"><input type="radio" name="grading_method" value="` + html.EscapeString(method) + `"` + checked + `><span>` + html.EscapeString(method) + `</span></label>`)
	}
	b.WriteString(`</fieldset><button type="submit">` + s.text(r, "save") + `</button></form>`)
	return template.HTML(b.String())
}

func markdownToHTML(src string) template.HTML {
	var b strings.Builder
	lines := strings.Split(src, "\n")
	inList := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			if inList {
				b.WriteString("</ul>")
				inList = false
			}
			b.WriteString("<h1>" + html.EscapeString(strings.TrimPrefix(line, "# ")) + "</h1>")
		} else if strings.HasPrefix(line, "## ") {
			if inList {
				b.WriteString("</ul>")
				inList = false
			}
			b.WriteString("<h2>" + html.EscapeString(strings.TrimPrefix(line, "## ")) + "</h2>")
		} else if strings.HasPrefix(line, "- ") {
			if !inList {
				b.WriteString("<ul>")
				inList = true
			}
			b.WriteString("<li>" + html.EscapeString(strings.TrimPrefix(line, "- ")) + "</li>")
		} else {
			if inList {
				b.WriteString("</ul>")
				inList = false
			}
			b.WriteString("<p>" + html.EscapeString(line) + "</p>")
		}
	}
	if inList {
		b.WriteString("</ul>")
	}
	return template.HTML(b.String())
}

func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return -1
	}, s)
	if s == "" {
		return "pixelbonus"
	}
	return s
}

func iconHome() template.HTML {
	return `<svg viewBox="0 0 24 24" width="28" height="28" aria-hidden="true"><path fill="#18bc9c" d="M12 3 3 10h2v10h5v-6h4v6h5V10h2z"/></svg>`
}

func iconUser() template.HTML {
	return `<svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true"><circle cx="12" cy="8" r="4" fill="#18bc9c"/><path d="M4 21c1.5-4 5-6 8-6s6.5 2 8 6" fill="none" stroke="#2c3e50" stroke-width="2"/></svg>`
}

func iconAcademicCap() template.HTML {
	return `<svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true"><path d="M5 11v6q7 5 14 0v-6" fill="#18bc9c" stroke="#2c3e50" stroke-width="1.5" stroke-linejoin="round"/><path d="m2 8 10-5 10 5-10 5z" fill="#18bc9c" stroke="#2c3e50" stroke-width="1.5" stroke-linejoin="round"/><path d="M22 8v8m-1 3 1-3 1 3" fill="none" stroke="#2c3e50" stroke-width="1.5" stroke-linecap="round"/></svg>`
}

func iconSettings() template.HTML {
	return `<svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true"><path fill="#18bc9c" stroke="#2c3e50" stroke-linejoin="round" d="m10 2 4 0 .6 3 2 .9 2.6-1.5 2 3.4-2.3 2v2.4l2.3 2-2 3.4-2.6-1.5-2 .9-.6 3h-4l-.6-3-2-.9-2.6 1.5-2-3.4 2.3-2v-2.4l-2.3-2 2-3.4L7.4 6l2-.9z"/><circle cx="12" cy="11.5" r="3" fill="white" stroke="#2c3e50"/></svg>`
}

func iconGrades() template.HTML {
	return `<svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true"><path d="M3 3v18h18" fill="none" stroke="#2c3e50" stroke-width="2"/><path d="M6 12h3v6H6zm5-5h3v11h-3zm5-4h3v15h-3z" fill="#18bc9c"/></svg>`
}

func iconNew() template.HTML {
	return `<svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true"><path d="M12 4v16M4 12h16" stroke="#18bc9c" stroke-width="3"/></svg>`
}

func iconPublic() template.HTML {
	return `<svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true"><g fill="none" stroke="#2c3e50" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><ellipse cx="12" cy="12" rx="4" ry="10"/><path d="M2 12h20M4 6h16M4 18h16"/></g></svg>`
}

func iconGitHub() template.HTML {
	return `<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><path fill="#2c3e50" d="M12 .5A12 12 0 0 0 8.2 23.9c.6.1.8-.2.8-.6v-2.3c-3.3.7-4-1.4-4-1.4-.6-1.4-1.3-1.8-1.3-1.8-1.1-.7.1-.7.1-.7 1.2.1 1.9 1.3 1.9 1.3 1.1 1.9 2.9 1.4 3.6 1.1.1-.8.4-1.4.8-1.7-2.7-.3-5.5-1.3-5.5-6A4.7 4.7 0 0 1 6.7 8c-.1-.3-.5-1.5.1-3.1 0 0 1-.3 3.2 1.2a11 11 0 0 1 5.8 0C18 4.6 19 5 19 5c.6 1.6.2 2.8.1 3.1a4.7 4.7 0 0 1 1.3 3.2c0 4.7-2.8 5.7-5.5 6 .4.4.8 1.1.8 2.2v3.2c0 .4.2.7.8.6A12 12 0 0 0 12 .5"/></svg>`
}
