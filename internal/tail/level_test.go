package tail

import "testing"

func TestDetectLevel(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{"FATAL: disk full", "fatal"},
		{"panic: runtime error", "fatal"},
		{"ERROR connecting to db", "error"},
		{"[err] file not found", "error"},
		{"critical: service down", "error"},
		{"WARN: retrying request", "warn"},
		{"[warning] slow query", "warn"},
		{"DEBUG handler called", "debug"},
		{"[trace] entering function", "debug"},
		{"INFO server started", "info"},
		{"user logged in successfully", "info"},
		{"", "info"},
	}
	for _, tc := range cases {
		got := detectLevel(tc.line)
		if got != tc.want {
			t.Errorf("detectLevel(%q) = %q, want %q", tc.line, got, tc.want)
		}
	}
}
