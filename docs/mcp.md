# Heimdall MCP Server

The `heimdall mcp` command starts a Model Context Protocol (MCP) server on stdin/stdout.
This lets AI agents like Claude query your Heimdall telemetry data as native tools —
no CLI invocation or output parsing required.

## Prerequisites

- Heimdall CLI installed
- A **Read-only API key** from the Heimdall dashboard

## Setup: Claude Desktop

Edit `~/.claude/claude_desktop_config.json` (create if missing):

```json
{
  "mcpServers": {
    "heimdall": {
      "command": "heimdall",
      "args": ["mcp"],
      "env": {
        "HEIMDALL_API_KEY": "hm_read_your_key_here",
        "HEIMDALL_BASE_URL": "https://api.yourdomain.com"
      }
    }
  }
}
```

Restart Claude Desktop. You'll see a hammer icon in the chat input indicating tools are available.

## Setup: Claude Code (CLI)

```bash
export HEIMDALL_API_KEY=hm_read_your_key_here
export HEIMDALL_BASE_URL=https://api.yourdomain.com
claude mcp add heimdall -- heimdall mcp
```

## Available tools

| Tool | Description |
|------|-------------|
| `get_overview` | Summary counts, avg duration, top failing events, latest errors |
| `list_events` | Paginated list of events with filters |
| `get_event` | Single event by ID with full metadata |
| `list_errors` | Paginated list of errors with filters |
| `get_error` | Single error by ID with **full stacktrace** |
| `list_performance` | Paginated list of performance records |
| `get_performance` | Single performance record by ID |

## Example prompts

- *"List the critical errors from Heimdall in the last 24 hours"*
- *"Read error `abc-123` from Heimdall and check what's happening in the code. Propose a fix."*
- *"Which endpoints are slowest according to Heimdall performance data this week?"*
- *"Summarize today's Heimdall overview and flag anything unusual"*

## Zero AI cost for Heimdall

The MCP server does not call any AI API. It only queries the Heimdall API and returns structured data. All AI usage costs belong to your own Claude account.
