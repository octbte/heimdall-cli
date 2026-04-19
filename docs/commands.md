# Heimdall CLI — Command Reference

## Global flags

| Flag | Description |
|------|-------------|
| `--profile <name>` | Config profile to use (default: `default_profile` from config file) |
| `--json` | Output as JSON instead of formatted table |

## heimdall configure

Interactive wizard to add a profile to `~/.heimdall/config.yaml`.

```bash
heimdall configure
heimdall configure list
```

## heimdall overview

Summary of events, errors, and performance for the configured project.

```bash
heimdall overview
heimdall overview --from 2026-04-01 --to 2026-04-19
heimdall overview --json
```

## heimdall events

```bash
heimdall events list [flags]
heimdall events get <id> [--json]
```

**Flags for `list`:**

| Flag | Description |
|------|-------------|
| `--level` | Filter: `debug`, `info`, `warning`, `error`, `critical` |
| `--status` | Filter: `success`, `error`, `warning`, `timeout`, `canceled` |
| `--name` | Filter by event name |
| `--environment` | Filter by environment ID |
| `--from` | Start time (ISO 8601 or YYYY-MM-DD) |
| `--to` | End time |
| `--limit` | Max results, default 50 |
| `--offset` | Pagination offset |

## heimdall errors

```bash
heimdall errors list [flags]
heimdall errors get <id> [--json]
```

**Flags for `list`:**

| Flag | Description |
|------|-------------|
| `--level` | Filter: `warning`, `error`, `critical` |
| `--fingerprint` | Filter by error fingerprint |
| `--endpoint` | Filter by API endpoint path |
| `--job` | Filter by job name |
| `--environment` | Filter by environment ID |
| `--from` / `--to` | Time range |
| `--limit` / `--offset` | Pagination |

## heimdall perf

```bash
heimdall perf list [flags]
heimdall perf get <id> [--json]
```

**Flags for `list`:**

| Flag | Description |
|------|-------------|
| `--metric` | Filter by metric name (e.g. `request_duration`) |
| `--target-type` | Filter: `http`, `job`, `function`, `task` |
| `--target-name` | Filter by target name |
| `--environment` | Filter by environment ID |
| `--from` / `--to` | Time range |
| `--limit` / `--offset` | Pagination |

## heimdall mcp

Starts the MCP server for AI agent integration. See [mcp.md](mcp.md).

```bash
heimdall mcp
```
