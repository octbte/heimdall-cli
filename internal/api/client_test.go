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

func TestGetLogLines(t *testing.T) {
	nextCursor := "cursor-abc"
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/log-lines" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("level"); got != "error" {
			t.Errorf("expected level=error, got %q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "100" {
			t.Errorf("expected limit=100, got %q", got)
		}
		json.NewEncoder(w).Encode(api.LogLinesResponse{
			Lines: []api.LogLineItem{
				{ID: "line-1", SourceName: "app.log", Level: "error", Message: "something went wrong"},
			},
			NextCursor: &nextCursor,
		})
	})

	resp, err := client.GetLogLines(api.ListLogLinesParams{Level: "error", Limit: 100})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(resp.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(resp.Lines))
	}
	if resp.Lines[0].ID != "line-1" {
		t.Errorf("expected line ID %q, got %q", "line-1", resp.Lines[0].ID)
	}
	if resp.NextCursor == nil || *resp.NextCursor != "cursor-abc" {
		t.Errorf("expected next_cursor %q, got %v", "cursor-abc", resp.NextCursor)
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
