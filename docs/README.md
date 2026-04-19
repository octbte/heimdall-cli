# Heimdall CLI

Query your Heimdall observability data from the terminal or connect it to an AI agent via MCP.

## Install

**macOS (Homebrew)**
```bash
brew install octobit/tap/heimdall
```

**Linux / macOS (curl)**
```bash
curl -sSL https://github.com/octobit/heimdall-cli/releases/latest/download/install.sh | sh
```

**Direct download**

Download the binary for your platform from [GitHub Releases](https://github.com/octobit/heimdall-cli/releases).

## Build from source

Requires Go 1.22+.

```bash
git clone https://github.com/octobit/heimdall-cli.git
cd heimdall-cli
go build -o heimdall .

# Optional: make it available system-wide
sudo mv heimdall /usr/local/bin/heimdall
```

## Quick start

1. Create a **Read-only API key** in the Heimdall dashboard (Settings > API Keys > New Key > Read-only)

2. Configure:
```bash
heimdall configure
```

3. Inspect your data:
```bash
heimdall overview
heimdall errors list --level critical
heimdall errors get <id>
```

4. Connect to Claude for AI-powered debugging — see [MCP Setup](mcp.md)

## Running locally (development)

### 1. Start the Heimdall backend

```bash
cd path/to/heimdall
docker compose up -d
```

### 2. Create a Read-only API key

You need a JWT token from a logged-in user first, then create a read key:

```bash
curl -X POST http://localhost:8000/api/v1/api-keys \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CLI local",
    "project_id": "YOUR_PROJECT_UUID",
    "scope": "read"
  }'
```

The response includes `raw_key` (starts with `hm_read_`). Copy it — it is shown only once.

### 3. Configure the CLI via environment variables

The fastest way for local development:

```bash
export HEIMDALL_API_KEY=hm_read_your_key_here
export HEIMDALL_BASE_URL=http://localhost:8000
```

Or use the interactive wizard to save a named profile:

```bash
heimdall configure
# Profile name [default]: local
# Read-only API key (hm_read_...): hm_read_your_key_here
# Heimdall base URL [https://api.heimdall.io]: http://localhost:8000
```

### 4. Query your data

```bash
heimdall overview
heimdall events list --limit 10
heimdall errors list --level error
heimdall errors get <error-id>
heimdall perf list --target-type http

# JSON output (useful for piping to jq)
heimdall errors list --json | jq '.items[0]'
```

### 5. Use as an MCP server with Claude Code

```bash
export HEIMDALL_API_KEY=hm_read_your_key_here
export HEIMDALL_BASE_URL=http://localhost:8000

claude mcp add heimdall -- heimdall mcp
```

Once added, you can ask Claude:
- *"List the critical errors from Heimdall"*
- *"Read error `abc-123` and propose a fix"*
- *"Which endpoints are slowest this week?"*

See [mcp.md](mcp.md) for full MCP setup instructions including Claude Desktop.

## Full command reference

See [commands.md](commands.md).
