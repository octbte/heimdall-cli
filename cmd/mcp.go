package cmd

import (
	mcpserver "github.com/octobit/heimdall-cli/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the MCP server for AI agent integration (stdin/stdout)",
	Long: `Starts the Heimdall MCP server on stdin/stdout.

This command is designed to be called by MCP clients such as Claude Desktop
or Claude Code, not by humans directly. The server exposes Heimdall telemetry
data as structured tools that AI agents can call natively.

Setup in Claude Desktop (~/.claude/claude_desktop_config.json):

  {
    "mcpServers": {
      "heimdall": {
        "command": "heimdall",
        "args": ["mcp"],
        "env": {
          "HEIMDALL_API_KEY": "hm_read_...",
          "HEIMDALL_BASE_URL": "https://api.yourdomain.com"
        }
      }
    }
  }

Setup in Claude Code:

  claude mcp add heimdall -- heimdall mcp`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := clientFromFlags()
		if err != nil {
			return err
		}
		return mcpserver.Serve(client)
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
