package tests

import (
	"os"
	"testing"
)

func TestAppArtifactsExist(t *testing.T) {
	for _, path := range []string{"../src/main.go", "../src/pixelbonus.service", "../README.md", "../implementation-notes.md"} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
}
