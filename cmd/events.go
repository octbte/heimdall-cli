package cmd

import (
	"fmt"
	"os"

	"github.com/octobit/heimdall-cli/internal/api"
	"github.com/octobit/heimdall-cli/internal/format"
	"github.com/spf13/cobra"
)

var (
	eventsLevel       string
	eventsStatus      string
	eventsName        string
	eventsEnvironment string
	eventsFrom        string
	eventsTo          string
	eventsLimit       int
	eventsOffset      int
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Query events captured by the Heimdall SDK",
}

var eventsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List events",
	Example: `  heimdall events list
  heimdall events list --level error --limit 20
  heimdall events list --from 2026-04-15 --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := clientFromFlags()
		if err != nil {
			return err
		}
		data, err := client.ListEvents(api.ListEventsParams{
			Level:       eventsLevel,
			Status:      eventsStatus,
			EventName:   eventsName,
			Environment: eventsEnvironment,
			From:        eventsFrom,
			To:          eventsTo,
			Limit:       eventsLimit,
			Offset:      eventsOffset,
		})
		if err != nil {
			return err
		}
		if jsonFlag {
			return format.JSON(os.Stdout, data)
		}
		fmt.Printf("Showing %d of %d events\n\n", len(data.Items), data.Pagination.Total)
		if len(data.Items) == 0 {
			fmt.Println("No events found.")
			return nil
		}
		rows := [][]string{}
		for _, e := range data.Items {
			rows = append(rows, []string{
				format.Truncate(e.ID, 36),
				format.LevelBadge(e.Level),
				e.Status,
				format.Truncate(e.EventName, 40),
				format.DerefInt(e.DurationMs, "-") + "ms",
				e.OccurredAt.Format("2006-01-02 15:04:05"),
			})
		}
		format.Table([]string{"ID", "LEVEL", "STATUS", "EVENT", "DURATION", "OCCURRED AT"}, rows)
		return nil
	},
}

var eventsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a specific event by ID",
	Args:  cobra.ExactArgs(1),
	Example: `  heimdall events get abc-123
  heimdall events get abc-123 --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := clientFromFlags()
		if err != nil {
			return err
		}
		e, err := client.GetEvent(args[0])
		if err != nil {
			return err
		}
		if jsonFlag {
			return format.JSON(os.Stdout, e)
		}
		fmt.Printf("Event: %s\n", e.ID)
		fmt.Printf("  Name:        %s\n", e.EventName)
		fmt.Printf("  Category:    %s\n", e.EventCategory)
		fmt.Printf("  Status:      %s\n", e.Status)
		fmt.Printf("  Level:       %s\n", format.LevelBadge(e.Level))
		fmt.Printf("  Duration:    %s ms\n", format.DerefInt(e.DurationMs, "-"))
		fmt.Printf("  Occurred at: %s\n", e.OccurredAt.Format("2006-01-02 15:04:05 UTC"))
		fmt.Printf("  Project:     %s\n", e.ProjectID)
		fmt.Printf("  Environment: %s\n", format.Deref(e.EnvironmentID, "-"))
		fmt.Printf("  Correlation: %s\n", format.Deref(e.CorrelationID, "-"))
		if len(e.Metadata) > 0 {
			fmt.Println("  Metadata:")
			for k, v := range e.Metadata {
				fmt.Printf("    %s: %v\n", k, v)
			}
		}
		if len(e.Tags) > 0 {
			fmt.Println("  Tags:")
			for k, v := range e.Tags {
				fmt.Printf("    %s: %v\n", k, v)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(eventsCmd)
	eventsCmd.AddCommand(eventsListCmd)
	eventsCmd.AddCommand(eventsGetCmd)

	eventsListCmd.Flags().StringVar(&eventsLevel, "level", "", "filter by level (debug|info|warning|error|critical)")
	eventsListCmd.Flags().StringVar(&eventsStatus, "status", "", "filter by status")
	eventsListCmd.Flags().StringVar(&eventsName, "name", "", "filter by event name")
	eventsListCmd.Flags().StringVar(&eventsEnvironment, "environment", "", "filter by environment ID")
	eventsListCmd.Flags().StringVar(&eventsFrom, "from", "", "start time filter (ISO 8601 or YYYY-MM-DD)")
	eventsListCmd.Flags().StringVar(&eventsTo, "to", "", "end time filter")
	eventsListCmd.Flags().IntVar(&eventsLimit, "limit", 50, "max results (1-200)")
	eventsListCmd.Flags().IntVar(&eventsOffset, "offset", 0, "pagination offset")
}
