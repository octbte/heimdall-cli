//go:build linux

package tail

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func generateSystemdUnit(execPath string, files []string, ingestKey string) string {
	execStart := execPath + " tail " + strings.Join(files, " ")
	return fmt.Sprintf(`[Unit]
Description=Heimdall log tail
After=network.target

[Service]
ExecStart=%s
Restart=on-failure
RestartSec=5s
Environment=HEIMDALL_INGEST_API_KEY=%s

[Install]
WantedBy=default.target
`, execStart, ingestKey)
}

func writeSystemdUnit(path, execPath string, files []string, ingestKey string) error {
	unit := generateSystemdUnit(execPath, files, ingestKey)
	return os.WriteFile(path, []byte(unit), 0644)
}

// InstallService installs a systemd user service for heimdall tail.
func InstallService(files []string, ingestKey string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating systemd user dir: %w", err)
	}
	unitPath := filepath.Join(dir, "heimdall-tail.service")

	if err := writeSystemdUnit(unitPath, self, files, ingestKey); err != nil {
		return fmt.Errorf("writing unit file: %w", err)
	}

	for _, subcmd := range [][]string{
		{"--user", "daemon-reload"},
		{"--user", "enable", "--now", "heimdall-tail"},
	} {
		if out, err := exec.Command("systemctl", subcmd...).CombinedOutput(); err != nil {
			return fmt.Errorf("systemctl %v: %w\n%s", subcmd, err, out)
		}
	}
	fmt.Printf("Service installed. Unit: %s\n", unitPath)
	return nil
}

// UninstallService stops and removes the systemd user service.
func UninstallService() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if out, err := exec.Command("systemctl", "--user", "disable", "--now", "heimdall-tail").CombinedOutput(); err != nil {
		fmt.Printf("warning: systemctl disable: %s\n", out)
	}

	unitPath := filepath.Join(home, ".config", "systemd", "user", "heimdall-tail.service")
	os.Remove(unitPath)
	fmt.Println("Service uninstalled.")
	return nil
}

// ServiceStatus prints the systemd service status.
func ServiceStatus() error {
	out, _ := exec.Command("systemctl", "--user", "status", "heimdall-tail").CombinedOutput()
	fmt.Print(string(out))
	return nil
}
