package tests

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"pixelbonus/src/i18n"
	"pixelbonus/src/logging"
	"pixelbonus/src/web"
)

func TestWebRequestsReachConfiguredLog(t *testing.T) {
	dir := t.TempDir()
	logger, err := logging.New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()
	catalogs, err := i18n.LoadCatalogs(filepath.Join("..", "src", "translations"))
	if err != nil {
		t.Fatal(err)
	}
	server, err := web.NewServer(web.Dependencies{
		TemplateDir: filepath.Join("..", "src", "templates"), Catalogs: catalogs, Logger: logger,
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := server.Routes()
	for _, path := range []string{"/?hash=secret-coupon", "/instructor/profile"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	}
	data, err := ioutil.ReadFile(filepath.Join(dir, "pixelbonus.log"))
	if err != nil {
		t.Fatal(err)
	}
	log := string(data)
	for _, want := range []string{"event=http_request", `method="GET"`, `path="/"`, `status="200"`, `path="/instructor/profile"`, `status="303"`, `duration=`} {
		if !strings.Contains(log, want) {
			t.Fatalf("log missing %s: %s", want, log)
		}
	}
	if strings.Contains(log, "secret-coupon") || strings.Contains(log, "hash=") {
		t.Fatal("request log includes query string")
	}
	// Reopening the logger must preserve previous requests.
	if err := logger.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := logging.New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	reopened.Log("INFO", "reopened", nil)
	reopened.Close()
	data, err = ioutil.ReadFile(filepath.Join(dir, "pixelbonus.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "event=http_request") || !strings.Contains(string(data), "event=reopened") {
		t.Fatal("logs were not appended")
	}
}
