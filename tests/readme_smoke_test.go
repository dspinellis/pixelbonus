package tests

import (
	"io/ioutil"
	"strings"
	"testing"
)

func TestReadmeContainsTestCommand(t *testing.T) {
	data, err := ioutil.ReadFile("../README.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "go test ./...") {
		t.Fatal("README missing test command")
	}
}
