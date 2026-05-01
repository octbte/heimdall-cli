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

A profile stores an optional read-only API key, an optional ingest API key,
and a base URL for one Heimdall project. At least one key is required.

  Read-only key (hm_read_...)  — needed for query commands (overview, events, errors, perf)
  Ingest key   (hm_live_...)  — needed for 'heimdall tail'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		r := bufio.NewReader(os.Stdin)

		profileName := prompt(r, "Profile name", "default")
		baseURL := prompt(r, "Heimdall base URL", "https://api.heimdall-ob.com")
		apiKey := prompt(r, "Read-only API key (optional, hm_read_...)", "")
		ingestKey := prompt(r, "Ingest API key (optional, hm_live_...)", "")

		if apiKey == "" && ingestKey == "" {
			return fmt.Errorf("at least one API key is required (read-only or ingest)")
		}

		if apiKey != "" {
			fmt.Print("Testing connection... ")
			client := api.NewClient(apiKey, baseURL)
			if _, err := client.GetOverview("", ""); err != nil {
				fmt.Println("failed")
				return fmt.Errorf("could not connect: %w", err)
			}
			fmt.Println("OK")
		}

		if err := config.Save("", profileName, config.Profile{
			APIKey:       apiKey,
			BaseURL:      baseURL,
			IngestAPIKey: ingestKey,
		}); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("\nProfile %q saved.\n", profileName)
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
