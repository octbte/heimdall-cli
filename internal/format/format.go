package format

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
)

func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func Table(headers []string, rows [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.SetBorder(false)
	table.SetColumnSeparator("  ")
	table.SetHeaderLine(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)
	for _, row := range rows {
		table.Append(row)
	}
	table.Render()
}

func LevelBadge(level string) string {
	switch strings.ToLower(level) {
	case "critical", "error":
		return fmt.Sprintf("\033[31m%s\033[0m", level)
	case "warning":
		return fmt.Sprintf("\033[33m%s\033[0m", level)
	default:
		return level
	}
}

func Truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

func Deref(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}

func DerefInt(i *int, fallback string) string {
	if i == nil {
		return fallback
	}
	return fmt.Sprintf("%d", *i)
}
