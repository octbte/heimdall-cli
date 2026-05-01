//go:build linux

package tail

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateSystemdUnit(t *testing.T) {
	unit := generateSystemdUnit(
		"/usr/local/bin/heimdall",
		[]string{"/var/log/app.log", "/var/log/worker.log"},
		"hm_live_testkey",
	)

	if !strings.Contains(unit, "ExecStart=/usr/local/bin/heimdall tail /var/log/app.log /var/log/worker.log") {
		t.Errorf("missing ExecStart in unit:\n%s", unit)
	}
	if !strings.Contains(unit, "HEIMDALL_INGEST_API_KEY=hm_live_testkey") {
		t.Errorf("missing HEIMDALL_INGEST_API_KEY in unit:\n%s", unit)
	}
	if !strings.Contains(unit, "Restart=on-failure") {
		t.Errorf("missing Restart=on-failure in unit:\n%s", unit)
	}
}

func TestWriteSystemdUnit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "heimdall-tail.service")

	err := writeSystemdUnit(path, "/usr/local/bin/heimdall", []string{"/var/log/app.log"}, "hm_live_key")
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "[Service]") {
		t.Error("unit file missing [Service] section")
	}
}
