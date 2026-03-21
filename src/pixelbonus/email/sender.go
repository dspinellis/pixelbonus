package email

import (
	"fmt"
	"io"
	"os/exec"
)

type Sender struct {
	commandPath string
	from        string
}

func NewSender(commandPath string, from string) Sender {
	return Sender{commandPath: commandPath, from: from}
}

func (s Sender) Send(to string, subject string, body string) error {
	if to == "" {
		return fmt.Errorf("recipient email is required")
	}
	cmd := exec.Command(s.commandPath, "-s", subject, "-a", "From: "+s.from, to)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("mail stdin: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start mail: %w", err)
	}
	if _, err := io.WriteString(stdin, body); err != nil {
		stdin.Close()
		cmd.Wait()
		return fmt.Errorf("write mail body: %w", err)
	}
	if err := stdin.Close(); err != nil {
		cmd.Wait()
		return fmt.Errorf("close mail stdin: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("mail command failed: %w", err)
	}
	return nil
}
