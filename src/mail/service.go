package mail

import (
	"context"
	"net/mail"
	"os/exec"
	"strings"
)

type Logger interface {
	Log(string, string, map[string]string)
}

type Runner func(context.Context, string, ...string) error

type Service struct {
	Logger Logger
	Run    Runner
}

func (s *Service) Send(ctx context.Context, to string, subject string, body string) error {
	if _, err := mail.ParseAddress(to); err != nil {
		return err
	}
	if s.Run == nil {
		s.Run = func(ctx context.Context, name string, args ...string) error {
			cmd := exec.CommandContext(ctx, name, args...)
			cmd.Stdin = strings.NewReader(body)
			return cmd.Run()
		}
	}
	err := s.Run(ctx, "mail", "-s", subject, "-a", "From: no-reply@pixelbonus.com", to)
	if err != nil && s.Logger != nil {
		s.Logger.Log("ERROR", "mail_send_failed", map[string]string{"to": to})
	}
	return err
}
