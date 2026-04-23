package cmd

import (
	"fmt"
	"os"

	"github.com/octobit/heimdall-cli/internal/api"
	"github.com/octobit/heimdall-cli/internal/format"
	"github.com/spf13/cobra"
)

var (
	errorsLevel       string
	errorsFingerprint string
	errorsEndpoint    string
	errorsJobName     string
	errorsEnvironment string
	errorsFrom        string
	errorsTo          string
	errorsLimit       int
	errorsOffset      int
)

var errorsCmd = &cobra.Command{
	Use:   "errors",
	Short: "Query errors and exceptions captured by the Heimdall SDK",
}

var errorsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List errors",
	Example: `  heimdall errors list
  heimdall errors list --level critical
  heimdall errors list --fingerprint "timeouterror:bank-transfer" --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := clientFromFlags()
		if err != nil {
			return err
		}
		data, err := client.ListErrors(api.ListErrorsParams{
			Level:       errorsLevel,
			Fingerprint: errorsFingerprint,
			Endpoint:    errorsEndpoint,
			JobName:     errorsJobName,
			Environment: errorsEnvironment,
			From:        errorsFrom,
			To:          errorsTo,
			Limit:       errorsLimit,
			Offset:      errorsOffset,
		})
		if err != nil {
			return err
		}
		if jsonFlag {
			return format.JSON(os.Stdout, data)
		}
		fmt.Printf("Showing %d of %d errors\n\n", len(data.Items), data.Pagination.Total)
		if len(data.Items) == 0 {
			fmt.Println("No errors found.")
			return nil
		}
		rows := [][]string{}
		for _, e := range data.Items {
			rows = append(rows, []string{
				format.Truncate(e.ID, 36),
				format.LevelBadge(e.Level),
				format.Truncate(e.ErrorType, 30),
				format.Truncate(e.Message, 50),
				format.Deref(e.Endpoint, "-"),
				e.OccurredAt.Format("2006-01-02 15:04:05"),
			})
		}
		format.Table([]string{"ID", "LEVEL", "TYPE", "MESSAGE", "ENDPOINT", "OCCURRED AT"}, rows)
		return nil
	},
}

var errorsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a specific error by ID, including full stacktrace",
	Args:  cobra.ExactArgs(1),
	Example: `  heimdall errors get abc-123
  heimdall errors get abc-123 --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := clientFromFlags()
		if err != nil {
			return err
		}
		e, err := client.GetError(args[0])
		if err != nil {
			return err
		}
		if jsonFlag {
			return format.JSON(os.Stdout, e)
		}
		fmt.Printf("Error: %s\n", e.ID)
		fmt.Printf("  Type:        %s\n", e.ErrorType)
		fmt.Printf("  Level:       %s\n", format.LevelBadge(e.Level))
		fmt.Printf("  Status:      %s\n", e.Status)
		fmt.Printf("  Message:     %s\n", e.Message)
		fmt.Printf("  Fingerprint: %s\n", format.Deref(e.Fingerprint, "-"))
		fmt.Printf("  Endpoint:    %s\n", format.Deref(e.Endpoint, "-"))
		fmt.Printf("  Job:         %s\n", format.Deref(e.JobName, "-"))
		fmt.Printf("  Occurred at: %s\n", e.OccurredAt.Format("2006-01-02 15:04:05 UTC"))
		fmt.Printf("  Project:     %s\n", e.ProjectID)
		fmt.Printf("  Correlation: %s\n", format.Deref(e.CorrelationID, "-"))
		fmt.Printf("  Trace ID:    %s\n", format.Deref(e.TraceID, "-"))
		fmt.Printf("  Span ID:     %s\n", format.Deref(e.SpanID, "-"))
		if e.Stacktrace != nil && *e.Stacktrace != "" {
			fmt.Printf("\nStacktrace:\n%s\n", *e.Stacktrace)
		}
		if len(e.Metadata) > 0 {
			fmt.Println("\nMetadata:")
			for k, v := range e.Metadata {
				fmt.Printf("  %s: %v\n", k, v)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(errorsCmd)
	errorsCmd.AddCommand(errorsListCmd)
	errorsCmd.AddCommand(errorsGetCmd)

	errorsListCmd.Flags().StringVar(&errorsLevel, "level", "", "filter by level (debug|info|warning|error|critical)")
	errorsListCmd.Flags().StringVar(&errorsFingerprint, "fingerprint", "", "filter by fingerprint")
	errorsListCmd.Flags().StringVar(&errorsEndpoint, "endpoint", "", "filter by endpoint path")
	errorsListCmd.Flags().StringVar(&errorsJobName, "job", "", "filter by job name")
	errorsListCmd.Flags().StringVar(&errorsEnvironment, "environment", "", "filter by environment ID")
	errorsListCmd.Flags().StringVar(&errorsFrom, "from", "", "start time filter (ISO 8601 or YYYY-MM-DD)")
	errorsListCmd.Flags().StringVar(&errorsTo, "to", "", "end time filter")
	errorsListCmd.Flags().IntVar(&errorsLimit, "limit", 50, "max results (1-200)")
	errorsListCmd.Flags().IntVar(&errorsOffset, "offset", 0, "pagination offset")
}
