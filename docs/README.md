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

## Full command reference

See [commands.md](commands.md).
