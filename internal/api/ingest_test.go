package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/octobit/heimdall-cli/internal/api"
)

func TestIngestLogLines(t *testing.T) {
	var received api.IngestLogLinesRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/ingest/log-lines" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "hm_live_test" {
			t.Errorf("unexpected key %s", r.Header.Get("X-API-Key"))
		}
		json.NewDecoder(r.Body).Decode(&received)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(api.IngestLogLinesResponse{Ingested: 2})
	}))
	defer srv.Close()

	client := api.NewIngestClient("hm_live_test", srv.URL)
	resp, err := client.IngestLogLines("app.log", []api.LogLineInput{
		{Message: "hello", Level: "info"},
		{Message: "boom", Level: "error", OccurredAt: "2026-04-29T10:00:00Z"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Ingested != 2 {
		t.Errorf("ingested=%d, want 2", resp.Ingested)
	}
	if received.SourceName != "app.log" {
		t.Errorf("source_name=%q, want app.log", received.SourceName)
	}
	if len(received.Lines) != 2 {
		t.Errorf("lines=%d, want 2", len(received.Lines))
	}
}

func TestIngestLogLinesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := api.NewIngestClient("hm_live_test", srv.URL)
	_, err := client.IngestLogLines("app.log", []api.LogLineInput{{Message: "x"}})
	if err == nil {
		t.Error("expected error on 503, got nil")
	}
}
