package cmd

import (
	"fmt"
	"os"

	"github.com/octobit/heimdall-cli/internal/api"
	"github.com/octobit/heimdall-cli/internal/format"
	"github.com/spf13/cobra"
)

var tracesCmd = &cobra.Command{
	Use:   "traces",
	Short: "Query distributed traces",
}

var tracesGetCmd = &cobra.Command{
	Use:   "get <trace_id>",
	Short: "Get all records for a distributed trace, displayed as a span tree",
	Args:  cobra.ExactArgs(1),
	Example: `  heimdall traces get abc123def456
  heimdall traces get abc123def456 --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := clientFromFlags()
		if err != nil {
			return err
		}
		data, err := client.GetTrace(args[0])
		if err != nil {
			return err
		}
		if jsonFlag {
			return format.JSON(os.Stdout, data)
		}
		printTraceTree(data)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tracesCmd)
	tracesCmd.AddCommand(tracesGetCmd)
}

func printTraceTree(data *api.TraceResponse) {
	fmt.Printf("Trace: %s\n", data.TraceID)
	fmt.Printf("%d records\n\n", len(data.Records))

	if len(data.Records) == 0 {
		fmt.Println("No records found.")
		return
	}

	// Build a set of all span IDs present in the response
	spanIDs := make(map[string]bool)
	for _, r := range data.Records {
		if r.SpanID != nil {
			spanIDs[*r.SpanID] = true
		}
	}

	// Build children map: parent_span_id -> []TraceRecord
	children := make(map[string][]api.TraceRecord)
	var roots []api.TraceRecord

	for _, r := range data.Records {
		if r.ParentSpanID == nil || !spanIDs[*r.ParentSpanID] {
			roots = append(roots, r)
		} else {
			key := *r.ParentSpanID
			children[key] = append(children[key], r)
		}
	}

	// Print each root with its subtree
	for i, root := range roots {
		isLast := i == len(roots)-1
		printTraceNode(root, children, "", isLast)
	}
}

func printTraceNode(r api.TraceRecord, children map[string][]api.TraceRecord, prefix string, isLast bool) {
	// Determine connector
	var connector string
	if prefix == "" {
		connector = ""
	} else if isLast {
		connector = "└─ "
	} else {
		connector = "├─ "
	}

	// Name: truncate to 40 chars, left-pad to 40 width
	name := format.Truncate(r.Name, 40)
	namePadded := fmt.Sprintf("%-40s", name)

	// Type: fixed 12 width
	typePadded := fmt.Sprintf("%-12s", r.Type)

	// Duration: fixed 7 width
	durStr := format.DerefInt(r.DurationMs, "-") + "ms"
	durPadded := fmt.Sprintf("%-7s", durStr)

	// Status: fixed 9 width
	statusStr := format.Deref(r.Status, "-")
	statusPadded := fmt.Sprintf("%-9s", statusStr)

	// Occurred at
	occurredAt := r.OccurredAt.Format("2006-01-02 15:04:05")

	fmt.Printf("%s%s%s  %s  %s  %s  %s\n",
		prefix,
		connector,
		namePadded,
		typePadded,
		durPadded,
		statusPadded,
		occurredAt,
	)

	// Determine child prefix extension
	var childPrefix string
	if prefix == "" {
		childPrefix = "   "
	} else if isLast {
		childPrefix = prefix + "   "
	} else {
		childPrefix = prefix + "│  "
	}

	// Print children (keyed by this record's span_id)
	var kids []api.TraceRecord
	if r.SpanID != nil {
		kids = children[*r.SpanID]
	}
	for i, child := range kids {
		isChildLast := i == len(kids)-1
		printTraceNode(child, children, childPrefix, isChildLast)
	}
}

