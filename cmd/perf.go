package cmd

import (
	"fmt"
	"os"

	"github.com/octobit/heimdall-cli/internal/api"
	"github.com/octobit/heimdall-cli/internal/format"
	"github.com/spf13/cobra"
)

var (
	perfMetric      string
	perfTargetType  string
	perfTargetName  string
	perfEnvironment string
	perfFrom        string
	perfTo          string
	perfLimit       int
	perfOffset      int
)

var perfCmd = &cobra.Command{
	Use:   "perf",
	Short: "Query performance records captured by the Heimdall SDK",
}

var perfListCmd = &cobra.Command{
	Use:   "list",
	Short: "List performance records",
	Example: `  heimdall perf list
  heimdall perf list --metric request_duration --target-type http
  heimdall perf list --from 2026-04-15 --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := clientFromFlags()
		if err != nil {
			return err
		}
		data, err := client.ListPerformance(api.ListPerfParams{
			MetricName:  perfMetric,
			TargetType:  perfTargetType,
			TargetName:  perfTargetName,
			Environment: perfEnvironment,
			From:        perfFrom,
			To:          perfTo,
			Limit:       perfLimit,
			Offset:      perfOffset,
		})
		if err != nil {
			return err
		}
		if jsonFlag {
			return format.JSON(os.Stdout, data)
		}
		fmt.Printf("Showing %d of %d records\n\n", len(data.Items), data.Pagination.Total)
		if len(data.Items) == 0 {
			fmt.Println("No performance records found.")
			return nil
		}
		rows := [][]string{}
		for _, p := range data.Items {
			rows = append(rows, []string{
				format.Truncate(p.ID, 36),
				format.LevelBadge(p.Level),
				p.MetricName,
				p.TargetType,
				format.Truncate(p.TargetName, 40),
				fmt.Sprintf("%d ms", p.DurationMs),
				p.OccurredAt.Format("2006-01-02 15:04:05"),
			})
		}
		format.Table([]string{"ID", "LEVEL", "METRIC", "TYPE", "TARGET", "DURATION", "OCCURRED AT"}, rows)
		return nil
	},
}

var perfGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a specific performance record by ID",
	Args:  cobra.ExactArgs(1),
	Example: `  heimdall perf get abc-123
  heimdall perf get abc-123 --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := clientFromFlags()
		if err != nil {
			return err
		}
		p, err := client.GetPerformance(args[0])
		if err != nil {
			return err
		}
		if jsonFlag {
			return format.JSON(os.Stdout, p)
		}
		fmt.Printf("Performance: %s\n", p.ID)
		fmt.Printf("  Metric:      %s\n", p.MetricName)
		fmt.Printf("  Target type: %s\n", p.TargetType)
		fmt.Printf("  Target:      %s\n", p.TargetName)
		fmt.Printf("  Duration:    %d ms\n", p.DurationMs)
		fmt.Printf("  Level:       %s\n", format.LevelBadge(p.Level))
		fmt.Printf("  Status:      %s\n", p.Status)
		fmt.Printf("  Occurred at: %s\n", p.OccurredAt.Format("2006-01-02 15:04:05 UTC"))
		fmt.Printf("  Project:     %s\n", p.ProjectID)
		fmt.Printf("  Correlation: %s\n", format.Deref(p.CorrelationID, "-"))
		fmt.Printf("  Trace ID:    %s\n", format.Deref(p.TraceID, "-"))
		fmt.Printf("  Span ID:     %s\n", format.Deref(p.SpanID, "-"))
		if len(p.Metadata) > 0 {
			fmt.Println("  Metadata:")
			for k, v := range p.Metadata {
				fmt.Printf("    %s: %v\n", k, v)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(perfCmd)
	perfCmd.AddCommand(perfListCmd)
	perfCmd.AddCommand(perfGetCmd)

	perfListCmd.Flags().StringVar(&perfMetric, "metric", "", "filter by metric name (e.g. request_duration)")
	perfListCmd.Flags().StringVar(&perfTargetType, "target-type", "", "filter by target type (http|job|function|task)")
	perfListCmd.Flags().StringVar(&perfTargetName, "target-name", "", "filter by target name")
	perfListCmd.Flags().StringVar(&perfEnvironment, "environment", "", "filter by environment ID")
	perfListCmd.Flags().StringVar(&perfFrom, "from", "", "start time filter (ISO 8601 or YYYY-MM-DD)")
	perfListCmd.Flags().StringVar(&perfTo, "to", "", "end time filter")
	perfListCmd.Flags().IntVar(&perfLimit, "limit", 50, "max results (1-200)")
	perfListCmd.Flags().IntVar(&perfOffset, "offset", 0, "pagination offset")
}
