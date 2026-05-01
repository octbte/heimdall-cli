//go:build darwin

package tail

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func generateLaunchdPlist(execPath string, files []string, ingestKey string) string {
	programArgs := fmt.Sprintf("\t\t<string>%s</string>\n\t\t<string>tail</string>", execPath)
	for _, f := range files {
		programArgs += fmt.Sprintf("\n\t\t<string>%s</string>", f)
	}

	logPath := launchLogPath()

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.heimdall.tail</string>
	<key>ProgramArguments</key>
	<array>
%s
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>EnvironmentVariables</key>
	<dict>
		<key>HEIMDALL_INGEST_API_KEY</key>
		<string>%s</string>
	</dict>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
</dict>
</plist>
`, programArgs, ingestKey, logPath, logPath)
}

func launchLogPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".heimdall", "tail.log")
}

// InstallService installs a launchd agent for heimdall tail.
func InstallService(files []string, ingestKey string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	if _, err := heimdallDir(); err != nil {
		return fmt.Errorf("creating heimdall dir: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating LaunchAgents dir: %w", err)
	}
	plistPath := filepath.Join(dir, "com.heimdall.tail.plist")

	plist := generateLaunchdPlist(self, files, ingestKey)
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return fmt.Errorf("writing plist: %w", err)
	}

	if out, err := exec.Command("launchctl", "load", plistPath).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load: %w\n%s", err, out)
	}
	fmt.Printf("Service installed. Plist: %s\n", plistPath)
	return nil
}

// UninstallService unloads and removes the launchd agent.
func UninstallService() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.heimdall.tail.plist")

	if out, err := exec.Command("launchctl", "unload", plistPath).CombinedOutput(); err != nil {
		fmt.Printf("warning: launchctl unload: %s\n", out)
	}
	os.Remove(plistPath)
	fmt.Println("Service uninstalled.")
	return nil
}

// ServiceStatus prints the launchd agent status.
func ServiceStatus() error {
	out, _ := exec.Command("launchctl", "list").Output()
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "heimdall") {
			fmt.Println(line)
		}
	}
	return nil
}
