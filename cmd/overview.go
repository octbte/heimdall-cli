package cmd

import (
	"fmt"
	"os"

	"github.com/octobit/heimdall-cli/internal/format"
	"github.com/spf13/cobra"
)

var (
	overviewFrom string
	overviewTo   string
)

var overviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "Show a summary of events, errors, and performance metrics",
	Example: `  heimdall overview
  heimdall overview --from 2026-04-01 --to 2026-04-19
  heimdall overview --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := clientFromFlags()
		if err != nil {
			return err
		}
		data, err := client.GetOverview(overviewFrom, overviewTo)
		if err != nil {
			return err
		}
		if jsonFlag {
			return format.JSON(os.Stdout, data)
		}

		s := data.Summary
		fmt.Printf("\nOverview\n")
		fmt.Printf("  Events:            %d\n", s.EventsCount)
		fmt.Printf("  Errors:            %d\n", s.ErrorsCount)
		fmt.Printf("  Performance:       %d\n", s.PerformanceCount)
		fmt.Printf("  Avg duration (ms): %.1f\n", s.AvgDurationMs)
		fmt.Printf("  Critical errors:   %d\n\n", s.CriticalErrorsCount)

		if len(data.TopFailingEvents) > 0 {
			fmt.Println("Top failing events:")
			rows := [][]string{}
			for _, e := range data.TopFailingEvents {
				rows = append(rows, []string{e.EventName, fmt.Sprintf("%d", e.Count)})
			}
			format.Table([]string{"EVENT", "COUNT"}, rows)
			fmt.Println()
		}

		if len(data.LatestErrors) > 0 {
			fmt.Println("Latest errors:")
			rows := [][]string{}
			for _, e := range data.LatestErrors {
				rows = append(rows, []string{
					format.Truncate(e.ID, 36),
					format.LevelBadge(e.Level),
					format.Truncate(e.ErrorType, 30),
					format.Truncate(e.Message, 50),
					e.OccurredAt.Format("2006-01-02 15:04:05"),
				})
			}
			format.Table([]string{"ID", "LEVEL", "TYPE", "MESSAGE", "OCCURRED AT"}, rows)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(overviewCmd)
	overviewCmd.Flags().StringVar(&overviewFrom, "from", "", "start time filter (ISO 8601 or YYYY-MM-DD)")
	overviewCmd.Flags().StringVar(&overviewTo, "to", "", "end time filter (ISO 8601 or YYYY-MM-DD)")
}
