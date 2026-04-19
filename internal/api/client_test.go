package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/octobit/heimdall-cli/internal/api"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*api.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return api.NewClient("hm_read_testkey", srv.URL), srv
}

func TestGetOverview(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "hm_read_testkey" {
			t.Error("missing X-API-Key header")
		}
		if r.URL.Path != "/api/v1/dashboard/overview" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		json.NewEncoder(w).Encode(api.OverviewResponse{})
	})

	_, err := client.GetOverview("", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestListEvents(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/events" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("level"); got != "error" {
			t.Errorf("expected level=error, got %q", got)
		}
		json.NewEncoder(w).Encode(api.EventsResponse{Items: []api.EventItem{}})
	})

	_, err := client.ListEvents(api.ListEventsParams{Level: "error", Limit: 50})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAPIErrorResponse(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "unauthorized",
				"message": "Invalid API key.",
			},
		})
	})

	_, err := client.GetOverview("", "")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}
