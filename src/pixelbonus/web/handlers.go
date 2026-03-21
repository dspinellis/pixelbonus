package web

import (
	"context"
	"encoding/csv"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"pixelbonus/src/pixelbonus/auth"
	"pixelbonus/src/pixelbonus/config"
	"pixelbonus/src/pixelbonus/coupon"
	"pixelbonus/src/pixelbonus/course"
	"pixelbonus/src/pixelbonus/grade"
	"pixelbonus/src/pixelbonus/i18n"
)

type server struct {
	cfg       config.Config
	catalog   i18n.Catalog
	authSvc   *auth.Service
	courseSvc *course.Service
	couponSvc *coupon.Service
	gradeSvc  *grade.Service
	templates *template.Template
}

type requestContext struct {
	Lang       string
	User       *auth.Instructor
	Session    *auth.Session
	AnonCSRF   string
	RequestURI string
}

type pageData struct {
	Title      string
	Lang       string
	User       *auth.Instructor
	Body       template.HTML
	CSRFToken  string
	AnonCSRF   string
	Content    string
	Message    string
	Error      string
	Form       map[string]string
	Courses    []course.Course
	Course     course.Course
	Rows       interface{}
	Privacy    string
	Lookup     coupon.CouponLookup
	QueryToken string
}

func NewServer(cfg config.Config, catalog i18n.Catalog, authSvc *auth.Service, courseSvc *course.Service, couponSvc *coupon.Service, gradeSvc *grade.Service) http.Handler {
	tmpl := template.Must(template.ParseGlob(filepath.Join(cfg.TemplatesPath, "*.html")))
	s := &server{
		cfg:       cfg,
		catalog:   catalog,
		authSvc:   authSvc,
		courseSvc: courseSvc,
		couponSvc: couponSvc,
		gradeSvc:  gradeSvc,
		templates: tmpl,
	}
	return http.HandlerFunc(s.ServeHTTP)
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := s.buildRequestContext(w, r)
	switch {
	case r.URL.Path == "/":
		s.home(w, r, ctx)
	case strings.HasPrefix(r.URL.Path, "/lang/"):
		s.changeLanguage(w, r)
	case r.URL.Path == "/instructor/login":
		s.login(w, r, ctx)
	case r.URL.Path == "/instructor/register":
		s.register(w, r, ctx)
	case r.URL.Path == "/instructor/verify":
		s.verify(w, r, ctx)
	case r.URL.Path == "/instructor/password/forgot":
		s.forgotPassword(w, r, ctx)
	case r.URL.Path == "/instructor/password/reset":
		s.resetPassword(w, r, ctx)
	case r.URL.Path == "/instructor/profile":
		s.profile(w, r, ctx)
	case r.URL.Path == "/instructor/profile/password":
		s.changePassword(w, r, ctx)
	case r.URL.Path == "/instructor/courses":
		s.courses(w, r, ctx)
	case r.URL.Path == "/instructor/courses/new":
		s.newCourse(w, r, ctx)
	case strings.HasPrefix(r.URL.Path, "/instructor/courses/"):
		s.courseSubroutes(w, r, ctx)
	case r.URL.Path == "/courses":
		http.NotFound(w, r)
	case strings.HasPrefix(r.URL.Path, "/courses/"):
		s.publicCourse(w, r, ctx)
	case r.URL.Path == "/redeem":
		s.redeem(w, r, ctx)
	case r.URL.Path == "/r":
		s.qrLanding(w, r, ctx)
	case strings.HasPrefix(r.URL.Path, "/downloads/"):
		s.download(w, r)
	case r.URL.Path == "/privacy":
		s.privacy(w, r, ctx)
	case strings.HasPrefix(r.URL.Path, "/assets/"):
		http.StripPrefix("/assets/", http.FileServer(http.Dir(s.cfg.AssetsPath))).ServeHTTP(w, r)
	case r.URL.Path == "/instructor/logout":
		s.logout(w, r, ctx)
	default:
		http.NotFound(w, r)
	}
}

func (s *server) buildRequestContext(w http.ResponseWriter, r *http.Request) requestContext {
	langCookie, _ := r.Cookie(s.cfg.LanguageCookie)
	lang := i18n.ResolveLanguage(cookieValue(langCookie), r.Header.Get("Accept-Language"))
	ctx := requestContext{Lang: lang, RequestURI: r.RequestURI}
	if sessionCookie, err := r.Cookie(s.cfg.SessionCookieName); err == nil {
		user, session, err := s.authSvc.InstructorBySession(r.Context(), sessionCookie.Value)
		if err == nil {
			ctx.User = &user
			ctx.Session = &session
		}
	}
	anonCookie, err := r.Cookie("pixelbonus_anon_csrf")
	if err != nil || anonCookie.Value == "" {
		ctx.AnonCSRF = fmt.Sprintf("%d", time.Now().UnixNano())
		http.SetCookie(w, &http.Cookie{Name: "pixelbonus_anon_csrf", Value: ctx.AnonCSRF, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: false})
	} else {
		ctx.AnonCSRF = anonCookie.Value
	}
	return ctx
}

func (s *server) home(w http.ResponseWriter, r *http.Request, ctx requestContext) {
	s.render(w, ctx, pageData{Title: s.catalog.T(ctx.Lang, "home.title"), Content: "home"})
}

func (s *server) changeLanguage(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/lang/")
	if code != "en" && code != "el" {
		http.NotFound(w, r)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: s.cfg.LanguageCookie, Value: code, Path: "/", MaxAge: 31536000, SameSite: http.SameSiteLaxMode})
	ref := r.Referer()
	if ref == "" {
		ref = "/"
	}
	http.Redirect(w, r, ref, http.StatusFound)
}

func (s *server) login(w http.ResponseWriter, r *http.Request, ctx requestContext) {
	if r.Method == http.MethodGet {
		s.render(w, ctx, pageData{Title: "Instructor Login", Content: "login", AnonCSRF: ctx.AnonCSRF})
		return
	}
	if !s.verifyAnonCSRF(r, ctx) {
		http.Error(w, "csrf validation failed", http.StatusForbidden)
		return
	}
	session, err := s.authSvc.Login(r.Context(), r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		s.render(w, ctx, pageData{Title: "Instructor Login", Content: "login", Error: err.Error(), AnonCSRF: ctx.AnonCSRF})
		return
	}
	http.SetCookie(w, &http.Cookie{Name: s.cfg.SessionCookieName, Value: session.ID, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, "/instructor/courses", http.StatusFound)
}

func (s *server) register(w http.ResponseWriter, r *http.Request, ctx requestContext) {
	if r.Method == http.MethodGet {
		s.render(w, ctx, pageData{Title: "Instructor Registration", Content: "register", AnonCSRF: ctx.AnonCSRF})
		return
	}
	if !s.verifyAnonCSRF(r, ctx) {
		http.Error(w, "csrf validation failed", http.StatusForbidden)
		return
	}
	_, err := s.authSvc.RegisterInstructor(r.Context(), auth.RegisterInstructorInput{
		FullName: r.FormValue("full_name"),
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
	})
	if err != nil {
		s.render(w, ctx, pageData{Title: "Instructor Registration", Content: "register", Error: err.Error(), AnonCSRF: ctx.AnonCSRF})
		return
	}
	s.render(w, ctx, pageData{Title: "Instructor Registration", Content: "message", Message: "Registration completed. Please verify your email."})
}

func (s *server) verify(w http.ResponseWriter, r *http.Request, ctx requestContext) {
	err := s.authSvc.VerifyInstructor(r.Context(), r.URL.Query().Get("token"))
	if err != nil {
		s.render(w, ctx, pageData{Title: "Email Verification", Content: "message", Error: err.Error()})
		return
	}
	s.render(w, ctx, pageData{Title: "Email Verification", Content: "message", Message: "Email verified. You can now log in."})
}

func (s *server) forgotPassword(w http.ResponseWriter, r *http.Request, ctx requestContext) {
	if r.Method == http.MethodGet {
		s.render(w, ctx, pageData{Title: "Password Recovery", Content: "forgot", AnonCSRF: ctx.AnonCSRF})
		return
	}
	if !s.verifyAnonCSRF(r, ctx) {
		http.Error(w, "csrf validation failed", http.StatusForbidden)
		return
	}
	_ = s.authSvc.RequestPasswordReset(r.Context(), r.FormValue("email"))
	s.render(w, ctx, pageData{Title: "Password Recovery", Content: "message", Message: "If the account exists, an email has been sent."})
}

func (s *server) resetPassword(w http.ResponseWriter, r *http.Request, ctx requestContext) {
	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.FormValue("token")
	}
	if r.Method == http.MethodGet {
		s.render(w, ctx, pageData{Title: "Reset Password", Content: "reset", QueryToken: token, AnonCSRF: ctx.AnonCSRF})
		return
	}
	if !s.verifyAnonCSRF(r, ctx) {
		http.Error(w, "csrf validation failed", http.StatusForbidden)
		return
	}
	err := s.authSvc.ResetPassword(r.Context(), token, r.FormValue("password"))
	if err != nil {
		s.render(w, ctx, pageData{Title: "Reset Password", Content: "reset", Error: err.Error(), QueryToken: token, AnonCSRF: ctx.AnonCSRF})
		return
	}
	s.render(w, ctx, pageData{Title: "Reset Password", Content: "message", Message: "Password updated."})
}

func (s *server) profile(w http.ResponseWriter, r *http.Request, ctx requestContext) {
	if ctx.User == nil || ctx.Session == nil {
		http.Redirect(w, r, "/instructor/login", http.StatusFound)
		return
	}
	s.render(w, ctx, pageData{Title: "Profile", Content: "profile", CSRFToken: ctx.Session.CSRFToken})
}

func (s *server) changePassword(w http.ResponseWriter, r *http.Request, ctx requestContext) {
	if ctx.User == nil || ctx.Session == nil {
		http.Redirect(w, r, "/instructor/login", http.StatusFound)
		return
	}
	if !s.verifySessionCSRF(r, ctx) {
		http.Error(w, "csrf validation failed", http.StatusForbidden)
		return
	}
	err := s.authSvc.ChangePassword(r.Context(), ctx.User.ID, r.FormValue("password"))
	if err != nil {
		s.render(w, ctx, pageData{Title: "Profile", Content: "profile", Error: err.Error(), CSRFToken: ctx.Session.CSRFToken})
		return
	}
	s.render(w, ctx, pageData{Title: "Profile", Content: "message", Message: "Password updated.", CSRFToken: ctx.Session.CSRFToken})
}

func (s *server) logout(w http.ResponseWriter, r *http.Request, ctx requestContext) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if ctx.Session != nil {
		if !s.verifySessionCSRF(r, ctx) {
			http.Error(w, "csrf validation failed", http.StatusForbidden)
			return
		}
		_ = s.authSvc.Logout(r.Context(), ctx.Session.ID)
	}
	http.SetCookie(w, &http.Cookie{Name: s.cfg.SessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *server) courses(w http.ResponseWriter, r *http.Request, ctx requestContext) {
	if ctx.User == nil || ctx.Session == nil {
		http.Redirect(w, r, "/instructor/login", http.StatusFound)
		return
	}
	list, err := s.courseSvc.ListCourses(r.Context(), ctx.User.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.render(w, ctx, pageData{Title: "Courses", Content: "courses", Courses: list, CSRFToken: ctx.Session.CSRFToken})
}

func (s *server) newCourse(w http.ResponseWriter, r *http.Request, ctx requestContext) {
	if ctx.User == nil || ctx.Session == nil {
		http.Redirect(w, r, "/instructor/login", http.StatusFound)
		return
	}
	if r.Method == http.MethodGet {
		s.render(w, ctx, pageData{Title: "New Course", Content: "course_form", CSRFToken: ctx.Session.CSRFToken})
		return
	}
	if !s.verifySessionCSRF(r, ctx) {
		http.Error(w, "csrf validation failed", http.StatusForbidden)
		return
	}
	_, err := s.courseSvc.CreateCourse(r.Context(), ctx.User.ID, course.CreateCourseInput{Name: r.FormValue("name"), GradingMethod: r.FormValue("grading_method")})
	if err != nil {
		s.render(w, ctx, pageData{Title: "New Course", Content: "course_form", Error: err.Error(), CSRFToken: ctx.Session.CSRFToken})
		return
	}
	http.Redirect(w, r, "/instructor/courses", http.StatusFound)
}

func (s *server) courseSubroutes(w http.ResponseWriter, r *http.Request, ctx requestContext) {
	if ctx.User == nil || ctx.Session == nil {
		http.Redirect(w, r, "/instructor/login", http.StatusFound)
		return
	}
	trimmed := strings.TrimPrefix(r.URL.Path, "/instructor/courses/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	courseID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch parts[1] {
	case "settings":
		s.courseSettings(w, r, ctx, courseID)
	case "coupons":
		if len(parts) == 3 && parts[2] == "new" {
			s.couponForm(w, r, ctx, courseID)
			return
		}
		s.queueCoupons(w, r, ctx, courseID)
	case "grades":
		if len(parts) == 3 && parts[2] == "csv" {
			s.gradeCSV(w, r, ctx, courseID)
			return
		}
		s.gradePage(w, r, ctx, courseID)
	default:
		http.NotFound(w, r)
	}
}

func (s *server) courseSettings(w http.ResponseWriter, r *http.Request, ctx requestContext, courseID int64) {
	if r.Method == http.MethodGet {
		c, err := s.courseSvc.GetCourse(r.Context(), ctx.User.ID, courseID)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		s.render(w, ctx, pageData{Title: "Course Settings", Content: "course_settings", Course: c, CSRFToken: ctx.Session.CSRFToken})
		return
	}
	if !s.verifySessionCSRF(r, ctx) {
		http.Error(w, "csrf validation failed", http.StatusForbidden)
		return
	}
	c, err := s.courseSvc.UpdateCourse(r.Context(), ctx.User.ID, courseID, course.UpdateCourseInput{Name: r.FormValue("name"), GradingMethod: r.FormValue("grading_method")})
	if err != nil {
		s.render(w, ctx, pageData{Title: "Course Settings", Content: "course_settings", Error: err.Error(), Course: course.Course{ID: courseID, Name: r.FormValue("name"), GradingMethod: r.FormValue("grading_method")}, CSRFToken: ctx.Session.CSRFToken})
		return
	}
	s.render(w, ctx, pageData{Title: "Course Settings", Content: "course_settings", Message: "Course updated.", Course: c, CSRFToken: ctx.Session.CSRFToken})
}

func (s *server) couponForm(w http.ResponseWriter, r *http.Request, ctx requestContext, courseID int64) {
	c, err := s.courseSvc.GetCourse(r.Context(), ctx.User.ID, courseID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	s.render(w, ctx, pageData{Title: "Generate Coupons", Content: "coupon_form", Course: c, CSRFToken: ctx.Session.CSRFToken})
}

func (s *server) queueCoupons(w http.ResponseWriter, r *http.Request, ctx requestContext, courseID int64) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	if !s.verifySessionCSRF(r, ctx) {
		http.Error(w, "csrf validation failed", http.StatusForbidden)
		return
	}
	quantity, _ := strconv.Atoi(r.FormValue("quantity"))
	_, err := s.couponSvc.QueueGeneration(r.Context(), ctx.User.ID, courseID, quantity)
	if err != nil {
		s.render(w, ctx, pageData{Title: "Generate Coupons", Content: "message", Error: err.Error(), CSRFToken: ctx.Session.CSRFToken})
		return
	}
	s.render(w, ctx, pageData{Title: "Generate Coupons", Content: "message", Message: "The coupons are being generated. You will be notified by email when they are ready."})
}

func (s *server) gradePage(w http.ResponseWriter, r *http.Request, ctx requestContext, courseID int64) {
	c, rows, err := s.gradeSvc.CourseGrades(r.Context(), ctx.User.ID, courseID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	s.render(w, ctx, pageData{Title: "Grades", Content: "grades", Course: c, Rows: rows, CSRFToken: ctx.Session.CSRFToken})
}

func (s *server) gradeCSV(w http.ResponseWriter, r *http.Request, ctx requestContext, courseID int64) {
	_, rows, err := s.gradeSvc.CourseGrades(r.Context(), ctx.User.ID, courseID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"student_id", "redemptions", "grade", "rank"})
	for _, row := range rows {
		_ = writer.Write([]string{row.StudentID, strconv.Itoa(row.Redemptions), strconv.Itoa(row.Grade), strconv.Itoa(row.Rank)})
	}
	writer.Flush()
}

func (s *server) publicCourse(w http.ResponseWriter, r *http.Request, ctx requestContext) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/courses/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) != 2 || parts[1] != "redemptions" {
		http.NotFound(w, r)
		return
	}
	courseID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	c, rows, err := s.courseSvc.PublicRedemptions(r.Context(), courseID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	s.render(w, ctx, pageData{Title: "Public Redemptions", Content: "public_redemptions", Course: c, Rows: rows})
}

func (s *server) qrLanding(w http.ResponseWriter, r *http.Request, ctx requestContext) {
	lookup, err := s.couponSvc.LookupCoupon(r.Context(), r.URL.Query().Get("c"))
	if err != nil {
		s.render(w, ctx, pageData{Title: "Redeem Coupon", Content: "message", Error: "Invalid coupon identifier (" + r.URL.Query().Get("c") + ")."})
		return
	}
	s.render(w, ctx, pageData{Title: "Redeem Coupon", Content: "redeem", Lookup: lookup, AnonCSRF: ctx.AnonCSRF})
}

func (s *server) redeem(w http.ResponseWriter, r *http.Request, ctx requestContext) {
	if r.Method == http.MethodGet {
		s.render(w, ctx, pageData{Title: "Redeem Coupon", Content: "redeem", AnonCSRF: ctx.AnonCSRF})
		return
	}
	if !s.verifyAnonCSRF(r, ctx) {
		http.Error(w, "csrf validation failed", http.StatusForbidden)
		return
	}
	result, err := s.couponSvc.RedeemByCode(r.Context(), r.FormValue("code"), r.FormValue("student_id"))
	if err == coupon.ErrInvalidCoupon {
		s.render(w, ctx, pageData{Title: "Redeem Coupon", Content: "message", Error: "Invalid coupon identifier (" + groupedForMessage(r.FormValue("code")) + ")."})
		return
	}
	if err != nil {
		s.render(w, ctx, pageData{Title: "Redeem Coupon", Content: "message", Error: err.Error()})
		return
	}
	switch result.Status {
	case "redeemed":
		s.render(w, ctx, pageData{Title: "Redeem Coupon", Content: "message", Message: fmt.Sprintf("Coupon %s for course %s and student identifier %s successfully redeemed.", result.CouponCode, result.CourseName, result.StudentID)})
	case "already_redeemed":
		s.render(w, ctx, pageData{Title: "Redeem Coupon", Content: "message", Message: fmt.Sprintf("Coupon %s for course %s has been already redeemed.", result.CouponCode, result.CourseName)})
	default:
		s.render(w, ctx, pageData{Title: "Redeem Coupon", Content: "message", Error: "Invalid coupon identifier (" + result.CouponCode + ")."})
	}
}

func (s *server) download(w http.ResponseWriter, r *http.Request) {
	nonce := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/downloads/"), ".pdf")
	file, err := s.couponSvc.FileByNonce(context.Background(), nonce)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	http.ServeFile(w, r, file.AbsolutePath)
}

func (s *server) privacy(w http.ResponseWriter, r *http.Request, ctx requestContext) {
	body, err := ioutil.ReadFile(s.cfg.PrivacyPolicyPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.render(w, ctx, pageData{Title: s.catalog.T(ctx.Lang, "privacy.title"), Content: "privacy", Privacy: string(body)})
}

func (s *server) render(w http.ResponseWriter, ctx requestContext, data pageData) {
	data.Lang = ctx.Lang
	data.User = ctx.User
	if data.CSRFToken == "" && ctx.Session != nil {
		data.CSRFToken = ctx.Session.CSRFToken
	}
	if data.AnonCSRF == "" {
		data.AnonCSRF = ctx.AnonCSRF
	}
	var body strings.Builder
	if err := s.templates.ExecuteTemplate(&body, data.Content, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data.Body = template.HTML(body.String())
	if err := s.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *server) verifySessionCSRF(r *http.Request, ctx requestContext) bool {
	return ctx.Session != nil && r.FormValue("csrf_token") == ctx.Session.CSRFToken
}

func (s *server) verifyAnonCSRF(r *http.Request, ctx requestContext) bool {
	return r.FormValue("csrf_token") != "" && r.FormValue("csrf_token") == ctx.AnonCSRF
}

func cookieValue(c *http.Cookie) string {
	if c == nil {
		return ""
	}
	return c.Value
}

func groupedForMessage(raw string) string {
	normalized, err := coupon.NormalizeCode(raw)
	if err != nil {
		return strings.TrimSpace(raw)
	}
	return normalizeGrouping(normalized)
}

func normalizeGrouping(normalized string) string {
	var parts []string
	for i := 0; i < len(normalized); i += 4 {
		parts = append(parts, normalized[i:i+4])
	}
	return strings.Join(parts, " ")
}
