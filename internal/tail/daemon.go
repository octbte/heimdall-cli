//go:build linux || darwin

package tail

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func heimdallDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".heimdall")
	return dir, os.MkdirAll(dir, 0700)
}

// Daemonize re-execs the current binary with the same args minus --daemon,
// detaches it from the terminal (new session), redirects output to
// ~/.heimdall/tail.log, and writes the PID to ~/.heimdall/tail.pid.
// The parent process prints a confirmation message and returns nil.
func Daemonize(files []string, extraArgs []string) error {
	dir, err := heimdallDir()
	if err != nil {
		return err
	}

	logPath := filepath.Join(dir, "tail.log")
	pidPath := filepath.Join(dir, "tail.pid")

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer logFile.Close()

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	args := append([]string{"tail"}, files...)
	args = append(args, extraArgs...)

	cmd := exec.Command(self, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting daemon: %w", err)
	}

	pid := cmd.Process.Pid
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0600); err != nil {
		return fmt.Errorf("writing PID file: %w", err)
	}

	fmt.Printf("Heimdall tail running in background (PID %d)\n", pid)
	fmt.Printf("Logs: %s\n", logPath)
	return nil
}

// StopDaemon reads ~/.heimdall/tail.pid and sends SIGTERM to the process.
func StopDaemon() error {
	dir, err := heimdallDir()
	if err != nil {
		return err
	}
	pidPath := filepath.Join(dir, "tail.pid")
	data, err := os.ReadFile(pidPath)
	if os.IsNotExist(err) {
		fmt.Println("No daemon running")
		return nil
	}
	if err != nil {
		return err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return fmt.Errorf("invalid PID file: %w", err)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process %d: %w", pid, err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("sending SIGTERM to %d: %w", pid, err)
	}

	os.Remove(pidPath)
	fmt.Printf("Sent SIGTERM to PID %d\n", pid)
	return nil
}

// DaemonLogPath returns the path to the daemon log file.
func DaemonLogPath() (string, error) {
	dir, err := heimdallDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tail.log"), nil
}
