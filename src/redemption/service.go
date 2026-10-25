package redemption

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"pixelbonus/src/database"
)

type Service struct {
	DB *sql.DB
}

type RedeemInput struct {
	Code      string
	StudentID string
}

type RedeemResult struct {
	Status     string
	Code       string
	CourseName string
	StudentID  string
}

type RedeemPageData struct {
	CourseName string
	Code       string
}

func NormalizeCode(code string) (string, error) {
	code = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), " ", ""))
	if len(code) != 40 {
		return "", errors.New("invalid code")
	}
	if _, err := hex.DecodeString(code); err != nil {
		return "", err
	}
	return code, nil
}

func EncodeTokenForURL(code string) string {
	raw, _ := hex.DecodeString(code)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func (s *Service) LookupRedeemPage(ctx context.Context, encodedToken string) (RedeemPageData, error) {
	raw, err := base64.RawURLEncoding.DecodeString(encodedToken)
	if err != nil {
		return RedeemPageData{}, err
	}
	code := strings.ToUpper(hex.EncodeToString(raw))
	var courseName string
	if err := s.DB.QueryRowContext(ctx, "SELECT c.name FROM QrCode q LEFT JOIN QrRequest request ON request.id = q.qrrequest_id JOIN QrSet qs ON qs.id = COALESCE(q.qrset_id, request.qrset_id) JOIN Course c ON c.id = qs.user_id WHERE q.code = ?", code).Scan(&courseName); err != nil {
		return RedeemPageData{}, err
	}
	return RedeemPageData{CourseName: courseName, Code: code}, nil
}

func (s *Service) Redeem(ctx context.Context, input RedeemInput, now time.Time) (RedeemResult, error) {
	code, err := NormalizeCode(input.Code)
	if err != nil {
		return RedeemResult{}, err
	}
	student := strings.TrimSpace(input.StudentID)
	if len(student) == 0 || len(student) > 20 {
		return RedeemResult{}, errors.New("invalid student identifier")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return RedeemResult{}, err
	}
	defer tx.Rollback()
	// Acquire the coupon row lock before reading Redemption. This serializes
	// redemptions even on the legacy schema, which has no unique redemption key.
	if _, err := tx.ExecContext(ctx, "UPDATE QrCode SET updated_at = updated_at WHERE code = ?", code); err != nil {
		return RedeemResult{}, err
	}
	var courseName string
	var redeemedBy sql.NullString
	err = tx.QueryRowContext(ctx, "SELECT c.name, r.participantNumber FROM QrCode q LEFT JOIN QrRequest request ON request.id = q.qrrequest_id JOIN QrSet qs ON qs.id = COALESCE(q.qrset_id, request.qrset_id) JOIN Course c ON c.id = qs.user_id LEFT JOIN Redemption r ON r.qrcode_code = q.code WHERE q.code = ?", code).Scan(&courseName, &redeemedBy)
	if err == sql.ErrNoRows {
		return RedeemResult{Status: "invalid", Code: code}, nil
	}
	if err != nil {
		return RedeemResult{}, err
	}
	if redeemedBy.Valid {
		return RedeemResult{Status: "already", Code: code, CourseName: courseName}, nil
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO Redemption(qrcode_code,participantNumber,created_at,updated_at) VALUES(?,?,?,?)", code, student, database.DateTime(now), database.DateTime(now)); err != nil {
		return RedeemResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return RedeemResult{}, err
	}
	return RedeemResult{Status: "success", Code: code, CourseName: courseName, StudentID: student}, nil
}
