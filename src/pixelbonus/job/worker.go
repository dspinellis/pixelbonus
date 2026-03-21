package job

import (
	"context"
	"io"
	"path/filepath"
	"time"

	"pixelbonus/src/pixelbonus/config"
	"pixelbonus/src/pixelbonus/coupon"
	"pixelbonus/src/pixelbonus/course"
	"pixelbonus/src/pixelbonus/email"
	"pixelbonus/src/pixelbonus/pdf"
)

type Worker struct {
	coupons *coupon.Service
	courses *course.Service
	mailer  email.Sender
	cfg     config.Config
	clock   func() time.Time
	random  io.Reader
}

func NewWorker(coupons *coupon.Service, courses *course.Service, mailer email.Sender, cfg config.Config, clock func() time.Time, random io.Reader) *Worker {
	return &Worker{coupons: coupons, courses: courses, mailer: mailer, cfg: cfg, clock: clock, random: random}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			w.runOnce(ctx)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) runOnce(ctx context.Context) {
	job, err := w.coupons.ClaimNextJob(ctx)
	if err != nil {
		return
	}
	tokens, err := coupon.GenerateTokens(w.random, job.Quantity)
	if err != nil {
		w.coupons.FailJob(ctx, job.ID, err.Error())
		return
	}
	nonce, err := coupon.GenerateFileNonce(w.random)
	if err != nil {
		w.coupons.FailJob(ctx, job.ID, err.Error())
		return
	}
	path := filepath.Join(w.cfg.PDFDir, nonce+".pdf")
	if err := pdf.RenderCoupons(path, job.CourseName, w.cfg.BaseURL, tokens); err != nil {
		w.coupons.FailJob(ctx, job.ID, err.Error())
		return
	}
	file := coupon.StoredFile{JobID: job.ID, Nonce: nonce, AbsolutePath: path, CreatedAt: w.clock().UTC()}
	if err := w.coupons.CompleteJob(ctx, job.ID, tokens, file); err != nil {
		w.coupons.FailJob(ctx, job.ID, err.Error())
		return
	}
	link := w.cfg.BaseURL + "/downloads/" + nonce + ".pdf"
	_ = w.mailer.Send(job.InstructorEmail, "Your PixelBonus coupons are ready", "Download the generated coupon PDF:\n\n"+link+"\n")
}
