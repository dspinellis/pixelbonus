package tokens

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pixelbonus/src/courses"
	"pixelbonus/src/database"
	"pixelbonus/src/pdf"
	"pixelbonus/src/redemption"
)

type Mailer interface {
	Send(context.Context, string, string, string) error
}

type PDFGenerator interface {
	GenerateCoupons(context.Context, string, string, []pdf.CouponPrintData) error
}

type Service struct {
	DB      *sql.DB
	Courses *courses.Service
	Mailer  Mailer
	Logger  interface {
		Log(string, string, map[string]string)
	}
	PDF      PDFGenerator
	BaseURL  string
	Now      func() time.Time
	Random   func([]byte) (int, error)
	mu       sync.Mutex
	inFlight map[int64]bool
}

type TokenSet struct {
	ID         int64
	CourseID   int64
	TokenCount int
	FileNonce  string
	FilePath   string
	Status     string
}

func (s *Service) init() {
	if s.Now == nil {
		s.Now = time.Now
	}
	if s.Random == nil {
		s.Random = rand.Read
	}
	if s.inFlight == nil {
		s.inFlight = map[int64]bool{}
	}
}

func (s *Service) CreateTokenSet(ctx context.Context, instructorID int64, courseID int64, count int, now time.Time) (TokenSet, error) {
	s.init()
	if count <= 0 || count%20 != 0 {
		return TokenSet{}, errors.New("invalid token count")
	}
	course, err := s.Courses.GetCourse(ctx, instructorID, courseID)
	if err != nil {
		return TokenSet{}, err
	}
	fileNonce, err := s.randomHex(16)
	if err != nil {
		return TokenSet{}, err
	}
	filePath := filepath.Join("/var/tmp/pixelbonus", fileNonce+".pdf")
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return TokenSet{}, err
	}
	defer tx.Rollback()
	stamp := database.DateTime(now)
	res, err := tx.ExecContext(ctx, "INSERT INTO QrSet(user_id,created_at,updated_at) VALUES(?,?,?)", courseID, stamp, stamp)
	if err != nil {
		return TokenSet{}, err
	}
	qrsetID, _ := res.LastInsertId()
	res, err = tx.ExecContext(ctx, "INSERT INTO QrRequest(qrset_id,status,quantity,created_at,updated_at) VALUES(?,?,?,?,?)", qrsetID, "pending", count, stamp, stamp)
	if err != nil {
		return TokenSet{}, err
	}
	setID, _ := res.LastInsertId()
	if _, err := tx.ExecContext(ctx, "INSERT INTO pixelbonus_pdf_jobs(request_id,file_nonce,file_path) VALUES(?,?,?)", setID, fileNonce, filePath); err != nil {
		return TokenSet{}, err
	}
	for i := 0; i < count; i++ {
		code, err := s.randomHex(20)
		if err != nil {
			return TokenSet{}, err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO QrCode(code,qrset_id,qrrequest_id,created_at,updated_at) VALUES(?,?,?,?,?)", code, qrsetID, setID, stamp, stamp); err != nil {
			return TokenSet{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return TokenSet{}, err
	}
	set := TokenSet{ID: setID, CourseID: course.ID, TokenCount: count, FileNonce: fileNonce, FilePath: filePath, Status: "pending"}
	go s.runJob(context.Background(), setID, filePath, course.Name)
	return set, nil
}

func (s *Service) ResumePendingJobs(ctx context.Context) error {
	s.init()
	rows, err := s.DB.QueryContext(ctx, "SELECT r.id, j.file_path FROM QrRequest r JOIN pixelbonus_pdf_jobs j ON j.request_id = r.id WHERE r.status = 'pending'")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			return err
		}
		go s.runJob(context.Background(), id, path, "")
	}
	return rows.Err()
}

func (s *Service) GetTokenByCode(ctx context.Context, normalizedCode string) (string, error) {
	return redemption.NormalizeCode(normalizedCode)
}

func (s *Service) GetDownloadPath(ctx context.Context, fileNonce string) (string, string, error) {
	var path, course string
	err := s.DB.QueryRowContext(ctx, "SELECT j.file_path, c.name FROM pixelbonus_pdf_jobs j JOIN QrRequest r ON r.id = j.request_id JOIN QrSet qs ON qs.id = r.qrset_id JOIN Course c ON c.id = qs.user_id WHERE j.file_nonce = ? AND r.status = 'ready'", fileNonce).Scan(&path, &course)
	return path, course, err
}

func (s *Service) randomHex(bytesLen int) (string, error) {
	buf := make([]byte, bytesLen)
	if _, err := s.Random(buf); err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(buf)), nil
}

func (s *Service) runJob(ctx context.Context, tokenSetID int64, filePath string, courseName string) {
	s.init()
	s.mu.Lock()
	if s.inFlight[tokenSetID] {
		s.mu.Unlock()
		return
	}
	s.inFlight[tokenSetID] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.inFlight, tokenSetID)
		s.mu.Unlock()
	}()
	rows, err := s.DB.QueryContext(ctx, "SELECT code FROM QrCode WHERE qrrequest_id = ? ORDER BY code", tokenSetID)
	if err != nil {
		s.fail(tokenSetID, err)
		return
	}
	defer rows.Close()
	var printable []pdf.CouponPrintData
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			s.fail(tokenSetID, err)
			return
		}
		printable = append(printable, pdf.CouponPrintData{Code: code, URL: s.BaseURL + "/r?c=" + redemption.EncodeTokenForURL(code)})
	}
	if err := rows.Err(); err != nil {
		s.fail(tokenSetID, err)
		return
	}
	rows.Close()
	if s.PDF == nil {
		s.PDF = &pdf.Service{}
	}
	if err := s.PDF.GenerateCoupons(ctx, filePath, s.BaseURL+"/r?c=", printable); err != nil {
		s.fail(tokenSetID, err)
		return
	}
	_, err = s.DB.ExecContext(ctx, "UPDATE QrRequest SET status = 'ready', updated_at = ? WHERE id = ?", database.DateTime(s.Now()), tokenSetID)
	if err != nil {
		s.fail(tokenSetID, err)
		return
	}
	var email string
	if err := s.DB.QueryRowContext(ctx, "SELECT u.email FROM QrRequest r JOIN QrSet qs ON qs.id = r.qrset_id JOIN Course c ON c.id = qs.user_id JOIN Users u ON u.id = c.user_id WHERE r.id = ?", tokenSetID).Scan(&email); err != nil {
		s.logJob("ERROR", "coupon_notification_failed", tokenSetID, err)
		return
	}
	if s.Mailer != nil {
		err := s.Mailer.Send(ctx, email, "PixelBonus coupons ready", fmt.Sprintf("%s/download/%s", s.BaseURL, filepath.Base(strings.TrimSuffix(filePath, ".pdf"))))
		if err != nil {
			s.logJob("ERROR", "coupon_notification_failed", tokenSetID, err)
			return
		}
		s.logJob("INFO", "coupon_notification_submitted", tokenSetID, nil)
	}
}

func (s *Service) fail(tokenSetID int64, err error) {
	s.logJob("ERROR", "coupon_generation_failed", tokenSetID, err)
	_, _ = s.DB.Exec("UPDATE pixelbonus_pdf_jobs SET failure_reason = ? WHERE request_id = ?", err.Error(), tokenSetID)
	_, _ = s.DB.Exec("UPDATE QrRequest SET status = 'failed', updated_at = ? WHERE id = ?", database.DateTime(s.Now()), tokenSetID)
}

func (s *Service) logJob(level, event string, id int64, err error) {
	if s.Logger == nil {
		return
	}
	fields := map[string]string{"request_id": fmt.Sprint(id)}
	if err != nil {
		fields["error"] = err.Error()
	}
	s.Logger.Log(level, event, fields)
}
