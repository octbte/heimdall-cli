package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/octobit/heimdall-cli/internal/api"
	"github.com/octobit/heimdall-cli/internal/config"
	"github.com/octobit/heimdall-cli/internal/format"
	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure a Heimdall profile interactively",
	Long: `Interactive wizard to add a profile to ~/.heimdall/config.yaml.

A profile stores a read-only API key and base URL for one Heimdall project.
Create a Read-only API key first in the Heimdall dashboard (Settings > API Keys).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		r := bufio.NewReader(os.Stdin)

		profileName := prompt(r, "Profile name", "default")
		apiKey := prompt(r, "Read-only API key (hm_read_...)", "")
		if apiKey == "" {
			return fmt.Errorf("API key is required")
		}
		baseURL := prompt(r, "Heimdall base URL", "https://api.heimdall-ob.com")

		fmt.Print("Testing connection... ")
		client := api.NewClient(apiKey, baseURL)
		if _, err := client.GetOverview("", ""); err != nil {
			fmt.Println("failed")
			return fmt.Errorf("could not connect: %w", err)
		}
		fmt.Println("OK")

		if err := config.Save("", profileName, config.Profile{APIKey: apiKey, BaseURL: baseURL}); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("\nProfile %q saved. Run 'heimdall overview' to get started.\n", profileName)
		return nil
	},
}

var configureListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		profiles, defaultProfile, err := config.ListProfiles("")
		if err != nil {
			return err
		}
		if len(profiles) == 0 {
			fmt.Println("No profiles configured. Run 'heimdall configure' to add one.")
			return nil
		}
		rows := [][]string{}
		for name, p := range profiles {
			def := ""
			if name == defaultProfile {
				def = "✓"
			}
			rows = append(rows, []string{name, def, format.Truncate(p.APIKey, 20), p.BaseURL})
		}
		format.Table([]string{"PROFILE", "DEFAULT", "API KEY", "BASE URL"}, rows)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configureCmd)
	configureCmd.AddCommand(configureListCmd)
}

func prompt(r *bufio.Reader, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}
