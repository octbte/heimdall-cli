//go:build linux || darwin

package cmd

import (
	"fmt"
	"os"

	"github.com/octobit/heimdall-cli/internal/config"
	"github.com/octobit/heimdall-cli/internal/tail"
	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage heimdall tail as a system service",
	Long: `Install, uninstall, and check the status of heimdall tail as a
boot-persistent system service.

Linux: manages a systemd user service (~/.config/systemd/user/heimdall-tail.service)
macOS: manages a launchd agent (~/Library/LaunchAgents/com.heimdall.tail.plist)`,
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install <file> [files...]",
	Short: "Install heimdall tail as a boot-persistent service",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("at least one file path is required")
		}

		profile, err := config.Load("", profileFlag)
		if err != nil {
			return err
		}
		ingestKey := profile.IngestAPIKey
		if envKey := os.Getenv("HEIMDALL_INGEST_API_KEY"); envKey != "" {
			ingestKey = envKey
		}
		if ingestKey == "" {
			return fmt.Errorf("ingest API key not configured — run 'heimdall configure' or set HEIMDALL_INGEST_API_KEY")
		}

		return tail.InstallService(args, ingestKey)
	},
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the heimdall tail system service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return tail.UninstallService()
	},
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of the heimdall tail system service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return tail.ServiceStatus()
	},
}

func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
	serviceCmd.AddCommand(serviceStatusCmd)
}
