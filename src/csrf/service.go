package csrf

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	Secret []byte
	Now    func() time.Time
	Random func([]byte) (int, error)
	MaxAge time.Duration
}

func New() *Service {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	return &Service{Secret: buf, Now: time.Now, Random: rand.Read, MaxAge: time.Hour}
}

func (s *Service) Issue(sessionID string) (string, error) {
	if s.Now == nil {
		s.Now = time.Now
	}
	if len(s.Secret) == 0 {
		tmp := make([]byte, 32)
		_, _ = rand.Read(tmp)
		s.Secret = tmp
	}
	ts := strconv.FormatInt(s.Now().UTC().Unix(), 10)
	payload := sessionID + ":" + ts
	mac := hmac.New(sha256.New, s.Secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString([]byte(payload + ":" + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)))), nil
}

func (s *Service) Verify(sessionID string, token string) bool {
	if s.Now == nil {
		s.Now = time.Now
	}
	if s.MaxAge == 0 {
		s.MaxAge = time.Hour
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return false
	}
	parts := strings.Split(string(raw), ":")
	if len(parts) != 3 || parts[0] != sessionID {
		return false
	}
	unix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return false
	}
	if s.Now().UTC().After(time.Unix(unix, 0).UTC().Add(s.MaxAge)) {
		return false
	}
	mac := hmac.New(sha256.New, s.Secret)
	mac.Write([]byte(parts[0] + ":" + parts[1]))
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(want), []byte(parts[2]))
}
