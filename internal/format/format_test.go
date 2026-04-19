package format_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/octobit/heimdall-cli/internal/format"
)

func TestJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	err := format.JSON(&buf, map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]string
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if out["key"] != "value" {
		t.Errorf("expected value, got %q", out["key"])
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		input string
		n     int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"hi", 2, "hi"},
	}
	for _, tc := range cases {
		got := format.Truncate(tc.input, tc.n)
		if got != tc.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tc.input, tc.n, got, tc.want)
		}
	}
}

func TestLevelBadge(t *testing.T) {
	if !strings.Contains(format.LevelBadge("error"), "error") {
		t.Error("LevelBadge should contain the level name")
	}
	if !strings.Contains(format.LevelBadge("critical"), "critical") {
		t.Error("LevelBadge should contain the level name")
	}
}
