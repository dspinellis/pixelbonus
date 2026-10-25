package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"pixelbonus/src/auth"
	"pixelbonus/src/config"
	"pixelbonus/src/courses"
	"pixelbonus/src/csrf"
	"pixelbonus/src/database"
	"pixelbonus/src/grades"
	"pixelbonus/src/i18n"
	"pixelbonus/src/logging"
	"pixelbonus/src/mail"
	"pixelbonus/src/public"
	"pixelbonus/src/redemption"
	"pixelbonus/src/session"
	"pixelbonus/src/tokens"
	"pixelbonus/src/web"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (runErr error) {
	initDB := flag.Bool("init-db", false, "create the MySQL database and application tables, then exit")
	flag.Parse()
	env := map[string]string{}
	for _, key := range []string{"PIXELBONUS_CONFIG", "PIXELBONUS_DB_DSN"} {
		env[key] = os.Getenv(key)
	}
	cfg, err := config.Load(env, nil)
	if err != nil {
		return err
	}
	logger, err := logging.New(cfg.LogDir, nil)
	if err != nil {
		return fmt.Errorf("initialize logging: %w", err)
	}
	defer logger.Close()
	defer func() {
		if runErr != nil {
			logger.Log("ERROR", "application_failed", map[string]string{"error": runErr.Error()})
		} else {
			logger.Log("INFO", "application_stopped", nil)
		}
	}()
	logger.Log("INFO", "application_starting", map[string]string{"port": strconv.Itoa(cfg.Port)})
	schema, err := ioutil.ReadFile("design/schema.sql")
	if err != nil {
		return err
	}
	db, err := database.OpenMySQL(cfg.DBDSN, string(schema), *initDB)
	if err != nil {
		return err
	}
	defer db.Close()
	if *initDB {
		return nil
	}
	catalogs, err := i18n.LoadCatalogs("src/translations")
	if err != nil {
		return err
	}
	mailer := &mail.Service{Logger: logger}
	authSvc := &auth.Service{DB: db, Mailer: mailer, BaseURL: cfg.BaseURL, Logger: logger}
	sessionStore := &session.Store{DB: db}
	csrfSvc := csrf.New()
	courseSvc := &courses.Service{DB: db}
	gradesSvc := &grades.Service{DB: db, Courses: courseSvc}
	redemptionSvc := &redemption.Service{DB: db}
	publicSvc := &public.Service{DB: db}
	tokenSvc := &tokens.Service{DB: db, Courses: courseSvc, Mailer: mailer, BaseURL: cfg.BaseURL, Logger: logger}
	if err := tokenSvc.ResumePendingJobs(context.Background()); err != nil {
		logger.Log("ERROR", "resume_coupon_jobs_failed", map[string]string{"error": err.Error()})
	}
	server, err := web.NewServer(web.Dependencies{
		PrivacyPolicyURLs: cfg.PrivacyPolicyURLs,
		TemplateDir:       cfg.TemplateDir,
		Catalogs:          catalogs,
		Auth:              authSvc,
		Sessions:          sessionStore,
		CSRF:              csrfSvc,
		Courses:           courseSvc,
		Tokens:            tokenSvc,
		Grades:            gradesSvc,
		Public:            publicSvc,
		Redemption:        redemptionSvc,
		DB:                db,
		Logger:            logger,
	})
	if err != nil {
		return err
	}
	httpServer := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.Port),
		Handler: server.Routes(),
	}
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigc)
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- httpServer.ListenAndServeTLS("/etc/letsencrypt/live/pixelbonus.com/fullchain.pem", "/etc/letsencrypt/live/pixelbonus.com/privkey.pem")
	}()
	select {
	case err := <-serverErrors:
		if err != http.ErrServerClosed {
			return fmt.Errorf("HTTPS server on %s: %w", httpServer.Addr, err)
		}
		return nil
	case <-sigc:
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return httpServer.Shutdown(ctx)
}
