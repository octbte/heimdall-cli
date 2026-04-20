# Heimdall MCP Server

The `heimdall mcp` command starts a [Model Context Protocol](https://modelcontextprotocol.io) (MCP) server on stdin/stdout.
MCP is an open standard — any compatible AI client can connect to it and query your Heimdall telemetry data as native tools, with no CLI invocation or output parsing required.

## Prerequisites

- Heimdall CLI installed
- A **Read-only API key** from the Heimdall dashboard (Settings > API Keys > New Key > Read-only)

## How it works

The MCP server speaks JSON-RPC over stdin/stdout. Your AI client starts the `heimdall mcp` process and communicates with it directly. Heimdall acts as a data source — it never calls any AI API itself.

## Setup by client

### Claude Desktop

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

Restart Claude Desktop. A hammer icon will appear in the chat input when tools are available.

### Claude Code (CLI)

```bash
export HEIMDALL_API_KEY=hm_read_your_key_here
export HEIMDALL_BASE_URL=https://api.yourdomain.com
claude mcp add heimdall -- heimdall mcp
```

### Cursor

Open Cursor Settings > MCP and add a new server:

```json
{
  "heimdall": {
    "command": "heimdall",
    "args": ["mcp"],
    "env": {
      "HEIMDALL_API_KEY": "hm_read_your_key_here",
      "HEIMDALL_BASE_URL": "https://api.yourdomain.com"
    }
  }
}
```

### Windsurf

Edit `~/.codeium/windsurf/mcp_config.json`:

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

### Other MCP-compatible clients

The server configuration is always the same pattern:

```json
{
  "command": "heimdall",
  "args": ["mcp"],
  "env": {
    "HEIMDALL_API_KEY": "hm_read_your_key_here",
    "HEIMDALL_BASE_URL": "https://api.yourdomain.com"
  }
}
```

Refer to your client's documentation for where to place this configuration.

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

The MCP server does not call any AI API. It only queries the Heimdall API and returns structured data. All AI usage costs belong to your own AI client account.
