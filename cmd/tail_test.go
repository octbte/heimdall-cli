package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/octobit/heimdall-cli/internal/api"
)

func TestTailCommandIngestsLines(t *testing.T) {
	var mu sync.Mutex
	var received []api.IngestLogLinesRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/ingest/log-lines" {
			var req api.IngestLogLinesRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decoding request body: %v", err)
			}
			mu.Lock()
			received = append(received, req)
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(api.IngestLogLinesResponse{Ingested: 1})
	}))
	defer srv.Close()

	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	cfgContent := strings.ReplaceAll(`
default_profile: default
profiles:
  default:
    api_key: hm_read_test
    base_url: BASE_URL
    ingest_api_key: hm_live_test
`, "BASE_URL", srv.URL)
	if err := os.WriteFile(cfgFile, []byte(cfgContent), 0600); err != nil {
		t.Fatal(err)
	}

	logFile := filepath.Join(dir, "app.log")
	if err := os.WriteFile(logFile, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		runTailWithConfig(cfgFile, logFile, 2*time.Second)
	}()

	time.Sleep(100 * time.Millisecond)
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("user logged in\n")
	f.WriteString("ERROR failed to connect\n")
	f.Close()

	<-done

	mu.Lock()
	defer mu.Unlock()

	if len(received) == 0 {
		t.Fatal("no ingest requests received")
	}

	var allLines []api.LogLineInput
	for _, r := range received {
		if r.SourceName != "app.log" {
			t.Errorf("source_name=%q, want app.log", r.SourceName)
		}
		allLines = append(allLines, r.Lines...)
	}

	foundError := false
	for _, l := range allLines {
		if l.Level == "error" {
			foundError = true
		}
	}
	if !foundError {
		t.Error("expected at least one line with level=error")
	}
}
