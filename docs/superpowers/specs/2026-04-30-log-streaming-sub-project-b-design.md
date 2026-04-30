# Log Streaming — Sub-project B: `heimdall tail` CLI Command

## Context

Sub-project B implements the `heimdall tail` command and `heimdall service` lifecycle management. It depends on Sub-project A (backend ingest endpoint at `POST /api/v1/ingest/log-lines`), which is complete.

| Sub-project | Scope |
|---|---|
| A | Backend: ingest endpoint, storage, query endpoint, retention cleanup |
| **B (this spec)** | `heimdall-cli tail` command — streams log lines to the ingest endpoint |
| C | UI: raw log viewer (real-time and historical) |
| D | AI analysis: on-demand and scheduled per log source |

---

## Goal

Allow users to tail one or more log files from an application server and stream them continuously to the Heimdall backend. Lines are buffered and sent in batches. The process can run in foreground, as a simple background daemon, or as a system service that survives reboots.

---

## 1. Commands

### `heimdall tail <files...>`

Tails one or more log files and streams them to the Heimdall ingest endpoint.

```
heimdall tail /var/log/app.log
heimdall tail /var/log/app.log /var/log/worker.log
heimdall tail --daemon /var/log/app.log /var/log/worker.log
heimdall tail --stop
heimdall tail --logs
```

**Flags:**

| Flag | Description |
|---|---|
| `--daemon` | Daemonize the process (detach from terminal) |
| `--stop` | Stop the running daemon (sends SIGTERM via PID file) |
| `--logs` | Print the path to the daemon log file |
| `--source-name` | Override `source_name` sent to the backend. Only valid when tailing a single file. |
| `--profile` | Inherited from root: which config profile to use |

### `heimdall service install <files...>`

Installs `heimdall tail` as a system service that starts automatically on boot.

```
heimdall service install /var/log/app.log /var/log/worker.log
heimdall service uninstall
heimdall service status
```

**Platform behavior:**
- **Linux**: generates and installs a systemd user service (`~/.config/systemd/user/heimdall-tail.service`), then runs `systemctl --user enable --now heimdall-tail`
- **macOS**: generates and installs a launchd agent (`~/Library/LaunchAgents/com.heimdall.tail.plist`), then runs `launchctl load`

**Prerequisite:** the `heimdall` binary must be in `$PATH`.

---

## 2. Authentication

The `tail` command uses a dedicated ingest API key (scope `ingest`), separate from the read key used by query commands.

### Config profile (extended)

```yaml
profiles:
  default:
    api_key: hm_read_...         # existing — used by events, errors, perf, etc.
    base_url: https://api.heimdall-ob.com
    ingest_api_key: hm_live_...  # new, optional — required for `tail`
```

### Resolution order

1. Env var `HEIMDALL_INGEST_API_KEY`
2. `ingest_api_key` field in the active profile
3. Error: `"ingest API key not configured. Run 'heimdall configure' or set HEIMDALL_INGEST_API_KEY"`

### `heimdall configure` update

After the existing read key prompt, adds:

```
Ingest API key (optional, required for 'heimdall tail'):
> hm_live_...
```

Leaving it blank skips without error.

---

## 3. File Tailing

One goroutine per file. All goroutines share a single batch queue that flushes to the backend.

### Startup behavior

- **First run** (no saved offset): seeks to the end of the file. Does not send existing content.
- **Subsequent runs**: resumes from the saved byte offset in `~/.heimdall/offsets/<sha256-of-abs-path>.json`.

### Log rotation detection

Every 1 second, each goroutine checks the file's inode. If the inode changed (file was rotated), it closes the old file descriptor, opens the new file from byte 0, and resets the saved offset.

### `source_name`

Defaults to the basename of the file path (e.g., `/var/log/app.log` → `app.log`). Overridable via `--source-name` when tailing a single file.

---

## 4. Level Detection

Applied to each line via case-insensitive regex, in priority order:

| Priority | Level sent | Patterns matched |
|---|---|---|
| 1 | `fatal` | `fatal`, `panic` |
| 2 | `error` | `error`, `err`, `critical` |
| 3 | `warn` | `warn`, `warning` |
| 4 | `debug` | `debug`, `trace` |
| 5 | `info` | fallback (no pattern matched) |

Match is checked anywhere in the line — no fixed format required.

---

## 5. Batching and Retry

### Flush triggers

A batch is sent when either condition is met:
- **500 lines** accumulated across all files, or
- **5 seconds** elapsed since the last flush

Each flush is one `POST /api/v1/ingest/log-lines` per `source_name` (files are grouped by source_name within the batch window).

### Backend unavailable

When a POST fails:
- The batch is held in memory.
- Retry with exponential backoff: 1s, 2s, 4s, 8s, … capped at 60s.
- In-memory buffer limit: **10,000 lines** total across all files. If the limit is reached, the oldest lines are dropped and a warning is logged: `"buffer full: dropped N lines"`.
- The file offset is **not advanced** past lines that have not been successfully sent.

---

## 6. Daemon Mode (`--daemon`)

```
heimdall tail --daemon /var/log/app.log
# → Heimdall tail running in background (PID 12345)
# → Logs: ~/.heimdall/tail.log
```

**Implementation:**
1. Double-fork (Unix) to fully detach from the terminal and session.
2. Redirect stdout/stderr to `~/.heimdall/tail.log`.
3. Write PID to `~/.heimdall/tail.pid`.
4. Parent process prints confirmation and exits.

**`--stop`:** reads `~/.heimdall/tail.pid`, sends `SIGTERM`. If no PID file exists: `"No daemon running"`.

**`--logs`:** prints `~/.heimdall/tail.log` (for use with `tail -f $(heimdall tail --logs)`).

**Limitation:** does not survive reboot. Use `heimdall service install` for production servers.

---

## 7. Service Management (`heimdall service`)

### Linux — systemd user service

**Generated file:** `~/.config/systemd/user/heimdall-tail.service`

```ini
[Unit]
Description=Heimdall log tail
After=network.target

[Service]
ExecStart=/path/to/heimdall tail /var/log/app.log /var/log/worker.log
Restart=on-failure
RestartSec=5s
Environment=HEIMDALL_INGEST_API_KEY=hm_live_...

[Install]
WantedBy=default.target
```

The `ExecStart` path is resolved via `os.Executable()`. The `HEIMDALL_INGEST_API_KEY` is embedded in the service file from the active profile at install time.

After writing the file:
```
systemctl --user daemon-reload
systemctl --user enable --now heimdall-tail
```

### macOS — launchd agent

**Generated file:** `~/Library/LaunchAgents/com.heimdall.tail.plist`

Standard launchd plist with `RunAtLoad = true`, `KeepAlive = true`, and the same `EnvironmentVariables` block for the ingest key.

After writing: `launchctl load ~/Library/LaunchAgents/com.heimdall.tail.plist`

### `uninstall`

- Linux: `systemctl --user disable --now heimdall-tail` + removes the `.service` file
- macOS: `launchctl unload` + removes the `.plist` file

### `status`

- Linux: runs `systemctl --user status heimdall-tail` and prints the output
- macOS: runs `launchctl list | grep heimdall` and prints the result

---

## 8. New Code Structure

**New files:**
- `cmd/tail.go` — `heimdall tail` Cobra command
- `cmd/service.go` — `heimdall service` Cobra command with `install`, `uninstall`, `status` subcommands
- `internal/tail/tailer.go` — core file-tailing logic (offset tracking, inode watching, level detection)
- `internal/tail/batcher.go` — line buffer, flush timer, retry with backoff
- `internal/tail/daemon.go` — double-fork daemonization, PID file management
- `internal/tail/service_linux.go` — systemd service file generation and installation
- `internal/tail/service_darwin.go` — launchd plist generation and installation
- `internal/tail/level.go` — level detection regex

**Modified files:**
- `internal/config/config.go` — add `IngestAPIKey string` to `Profile` struct
- `internal/api/client.go` — add `IngestLogLines(sourceName string, lines []LogLineInput) error`
- `internal/api/types.go` — add `LogLineInput`, `IngestLogLinesRequest`, `IngestLogLinesResponse`
- `cmd/configure.go` — add ingest key prompt
- `cmd/root.go` — register `tailCmd` and `serviceCmd`

---

## 9. Testing

### Unit tests (`internal/tail/`)

- `TestLevelDetect` — one case per level including fallback to `info`
- `TestOffsetPersistence` — write offset, restart, verify resume position
- `TestInodeRotationDetected` — mock inode change, verify file reopened from byte 0
- `TestBatchFlushBySize` — 500 lines buffered → flush triggered
- `TestBatchFlushByTimer` — < 500 lines, 5 seconds elapse → flush triggered
- `TestRetryBackoff` — first POST fails, second succeeds → verify retry with delay
- `TestBufferOverflow` — backend down + 10,001 lines → oldest line dropped, warning logged

### Integration tests (`cmd/tail_test.go`)

- Start a mock HTTP server that records ingest requests
- Write lines to a temp file, run the tailer, assert the mock receives correct batches
- Verify `source_name` = basename of the file
- Verify level detection on sample lines

### Not tested automatically

- `--daemon` (requires fork, tested manually)
- `heimdall service install/uninstall/status` (requires systemd/launchd, tested manually on target platform)

---

## Out of Scope

- Sub-project C: UI log viewer
- Sub-project D: AI analysis
- WebSocket / SSE streaming
- `--follow=name` (inotify-based) — current implementation uses polling every 1s
- Windows support
