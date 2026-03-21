package tests

import (
	"testing"

	"pixelbonus/src/pixelbonus/i18n"
)

func TestResolveLanguage(t *testing.T) {
	if got := i18n.ResolveLanguage("", "el-GR,el;q=0.9,en;q=0.8"); got != "el" {
		t.Fatalf("expected el, got %s", got)
	}
	if got := i18n.ResolveLanguage("", "fr-FR"); got != "en" {
		t.Fatalf("expected en, got %s", got)
	}
}
