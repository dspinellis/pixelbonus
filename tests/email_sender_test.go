package tests

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"pixelbonus/src/pixelbonus/email"
)

func TestEmailSenderWritesBody(t *testing.T) {
	out := filepath.Join(t.TempDir(), "mail.txt")
	script := writeMailScript(t, out)
	sender := email.NewSender(script, "no-reply@pixelbonus.com")
	if err := sender.Send("user@example.com", "subject", "body"); err != nil {
		t.Fatal(err)
	}
	data, err := ioutil.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "body" {
		t.Fatalf("unexpected body: %q", string(data))
	}
}
