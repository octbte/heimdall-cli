package cmd

import (
	"fmt"
	"os"

	"github.com/octobit/heimdall-cli/internal/api"
	"github.com/octobit/heimdall-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	profileFlag string
	jsonFlag    bool
)

var rootCmd = &cobra.Command{
	Use:   "heimdall",
	Short: "Heimdall CLI — query observability data from your terminal or AI agent",
	Long: `Heimdall CLI lets you inspect events, errors, and performance metrics
captured by the Heimdall SDK directly from your terminal.

Configure once with 'heimdall configure', then query your project data.
Use 'heimdall mcp' to start the MCP server for AI agent integration.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&profileFlag, "profile", "", "config profile to use (default: default_profile from config file)")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "output as JSON instead of table")
}

func clientFromFlags() (*api.Client, error) {
	profile, err := config.Load("", profileFlag)
	if err != nil {
		return nil, err
	}
	if profile.BaseURL == "" {
		return nil, fmt.Errorf("base_url is not set in profile — run 'heimdall configure'")
	}
	return api.NewClient(profile.APIKey, profile.BaseURL), nil
}
