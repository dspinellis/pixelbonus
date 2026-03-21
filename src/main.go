package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"pixelbonus/src/pixelbonus/auth"
	"pixelbonus/src/pixelbonus/config"
	"pixelbonus/src/pixelbonus/coupon"
	"pixelbonus/src/pixelbonus/course"
	"pixelbonus/src/pixelbonus/database"
	"pixelbonus/src/pixelbonus/email"
	"pixelbonus/src/pixelbonus/grade"
	"pixelbonus/src/pixelbonus/i18n"
	"pixelbonus/src/pixelbonus/job"
	"pixelbonus/src/pixelbonus/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Open(cfg, cfg.SchemaPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	catalog := i18n.NewCatalog()
	mailer := email.NewSender(cfg.MailCommandPath, cfg.MailFrom)
	authSvc := auth.NewService(db, mailer, cfg.BaseURL, time.Now, strings.NewReader(strings.Repeat("a", 4096)))
	courseSvc := course.NewService(db, time.Now)
	couponSvc := coupon.NewService(db, time.Now, strings.NewReader(strings.Repeat("b", 4096)))
	gradeSvc := grade.NewService(db)
	worker := job.NewWorker(couponSvc, courseSvc, mailer, cfg, time.Now, strings.NewReader(strings.Repeat("c", 4096)))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go worker.Run(ctx)

	server := &http.Server{
		Addr:    cfg.Address,
		Handler: web.NewServer(cfg, catalog, authSvc, courseSvc, couponSvc, gradeSvc),
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		server.Shutdown(shutdownCtx)
	}()

	log.Fatal(server.ListenAndServeTLS(cfg.TLSCertPath, cfg.TLSKeyPath))
}
