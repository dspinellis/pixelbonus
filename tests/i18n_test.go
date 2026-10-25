package tests

import (
	"path/filepath"
	"testing"

	"pixelbonus/src/i18n"
)

func TestCatalogsLoadAndTranslate(t *testing.T) {
	c, err := i18n.LoadCatalogs(filepath.Join("..", "src", "translations"))
	if err != nil {
		t.Fatal(err)
	}
	got := c.T("el", "home_heading", nil)
	if got == "" || got == "Your participation counts!" {
		t.Fatalf("unexpected translation: %q", got)
	}
}

func TestBrowserLanguagePreferences(t *testing.T) {
	for _, tc := range []struct{ name, explicit, cookie, header, want string }{
		{"Greek region", "", "", "el-GR,el;q=0.9,en;q=0.8", "el"},
		{"supported fallback", "", "", "fr-FR,el-GR;q=0.8,en;q=0.2", "el"},
		{"quality order", "", "", "en;q=0.2,el;q=0.9", "el"},
		{"English preference", "", "", "el;q=0.2,en-US;q=0.9", "en"},
		{"zero quality", "", "", "el;q=0,en;q=0.5", "en"},
		{"invalid quality", "", "", "el;q=invalid,en", "en"},
		{"manual choice", "en", "el", "el-GR", "en"},
		{"saved choice", "", "el", "en-US", "el"},
		{"invalid choices", "unknown", "english", "fr,el;q=0.8", "el"},
		{"default", "", "", "de,fr", "en"},
		{"equal priority", "", "", "el;q=0.8,en;q=0.8", "el"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := i18n.ResolveLanguage(tc.explicit, tc.cookie, tc.header); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}
