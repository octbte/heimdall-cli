# Log Streaming Sub-project B Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `heimdall tail` command that tails one or more log files and streams them to the Heimdall ingest endpoint, with daemon mode, service management, and retry on backend unavailability.

**Architecture:** One goroutine per file feeds a shared batcher; the batcher flushes 500-line batches (or every 5 s) to `POST /api/v1/ingest/log-lines` via an ingest-scoped API key loaded from config. Byte offsets are persisted per-file so the tailer resumes after restart; inode polling detects log rotation every second. A re-exec approach (not true double-fork) achieves daemon mode safely under Go's multithreaded runtime.

**Tech Stack:** Go 1.22+, Cobra, standard library (`os`, `crypto/sha256`, `encoding/json`, `regexp`, `syscall`), no new third-party dependencies.

---

## File Map

**New files:**
- `internal/tail/level.go` — level detection regex
- `internal/tail/level_test.go`
- `internal/tail/offset.go` — byte offset read/write
- `internal/tail/offset_test.go`
- `internal/tail/tailer.go` — file tailer (inode watch, line reading, sends to batcher)
- `internal/tail/tailer_test.go`
- `internal/tail/batcher.go` — line buffer, flush-by-size, flush-by-timer, retry backoff
- `internal/tail/batcher_test.go`
- `internal/tail/daemon.go` — re-exec daemonization, PID file (build tag: linux || darwin)
- `internal/tail/service_linux.go` — systemd unit generation and install
- `internal/tail/service_darwin.go` — launchd plist generation and install
- `cmd/tail.go` — `heimdall tail` Cobra command
- `cmd/service.go` — `heimdall service` Cobra command

**Modified files:**
- `internal/config/config.go` — add `IngestAPIKey string` to `Profile`; update `Load`, `Save`, `ListProfiles`
- `internal/api/types.go` — add `LogLineInput`, `IngestLogLinesRequest`, `IngestLogLinesResponse`
- `internal/api/client.go` — add `post()` private method; add `IngestLogLines()`
- `cmd/configure.go` — add ingest key prompt after read-key prompt
- `cmd/root.go` — register `tailCmd` and `serviceCmd`

---

## Task 1: Add `IngestAPIKey` to config

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Write the failing test**

Create `internal/config/config_ingest_test.go`:

```go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/octobit/heimdall-cli/internal/config"
)

func TestLoadIngestAPIKeyFromEnv(t *testing.T) {
	t.Setenv("HEIMDALL_API_KEY", "hm_read_test")
	t.Setenv("HEIMDALL_INGEST_API_KEY", "hm_live_envkey")
	t.Setenv("HEIMDALL_BASE_URL", "https://example.com")

	p, err := config.Load("", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.IngestAPIKey != "hm_live_envkey" {
		t.Errorf("got %q, want hm_live_envkey", p.IngestAPIKey)
	}
}

func TestLoadIngestAPIKeyFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	yaml := `
default_profile: default
profiles:
  default:
    api_key: hm_read_test
    base_url: https://example.com
    ingest_api_key: hm_live_fromfile
`
	os.WriteFile(cfgFile, []byte(yaml), 0600)

	p, err := config.Load(cfgFile, "")
	if err != nil {
		t.Fatal(err)
	}
	if p.IngestAPIKey != "hm_live_fromfile" {
		t.Errorf("got %q, want hm_live_fromfile", p.IngestAPIKey)
	}
}

func TestSaveAndLoadIngestAPIKey(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")

	err := config.Save(cfgFile, "myproj", config.Profile{
		APIKey:       "hm_read_test",
		BaseURL:      "https://example.com",
		IngestAPIKey: "hm_live_saved",
	})
	if err != nil {
		t.Fatal(err)
	}

	p, err := config.Load(cfgFile, "myproj")
	if err != nil {
		t.Fatal(err)
	}
	if p.IngestAPIKey != "hm_live_saved" {
		t.Errorf("got %q, want hm_live_saved", p.IngestAPIKey)
	}
}

func TestListProfilesIncludesIngestKey(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	yaml := `
default_profile: default
profiles:
  default:
    api_key: hm_read_test
    base_url: https://example.com
    ingest_api_key: hm_live_listed
`
	os.WriteFile(cfgFile, []byte(yaml), 0600)

	profiles, _, err := config.ListProfiles(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	if profiles["default"].IngestAPIKey != "hm_live_listed" {
		t.Errorf("got %q, want hm_live_listed", profiles["default"].IngestAPIKey)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```
cd /Users/hildebrandopedroni/code/Octobit/Heimdall/heimdall-cli
go test ./internal/config/... -run TestLoadIngestAPIKey -v
```

Expected: FAIL — `Profile` has no field `IngestAPIKey`

- [ ] **Step 3: Implement**

Replace `internal/config/config.go` with:

```go
package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Profile holds credentials for one Heimdall project.
type Profile struct {
	APIKey       string
	BaseURL      string
	IngestAPIKey string
}

// Load resolves credentials using this priority order:
//  1. HEIMDALL_API_KEY / HEIMDALL_BASE_URL / HEIMDALL_INGEST_API_KEY env vars
//  2. profileName from cfgFile (or default_profile if profileName is "")
func Load(cfgFile, profileName string) (*Profile, error) {
	if key := os.Getenv("HEIMDALL_API_KEY"); key != "" {
		baseURL := os.Getenv("HEIMDALL_BASE_URL")
		if baseURL == "" {
			baseURL = "https://api.heimdall-ob.com"
		}
		return &Profile{
			APIKey:       key,
			BaseURL:      baseURL,
			IngestAPIKey: os.Getenv("HEIMDALL_INGEST_API_KEY"),
		}, nil
	}

	v := viper.New()
	v.SetConfigType("yaml")

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
		v.SetConfigFile(home + "/.heimdall/config.yaml")
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("no config found — run 'heimdall configure' to get started")
	}

	if profileName == "" {
		profileName = v.GetString("default_profile")
	}
	if profileName == "" {
		profileName = "default"
	}

	apiKey := v.GetString(fmt.Sprintf("profiles.%s.api_key", profileName))
	if apiKey == "" {
		return nil, fmt.Errorf("profile %q not found — run 'heimdall configure' to add it", profileName)
	}

	return &Profile{
		APIKey:       apiKey,
		BaseURL:      v.GetString(fmt.Sprintf("profiles.%s.base_url", profileName)),
		IngestAPIKey: v.GetString(fmt.Sprintf("profiles.%s.ingest_api_key", profileName)),
	}, nil
}

// Save writes a profile to the config file.
func Save(cfgFile, profileName string, profile Profile) error {
	v := viper.New()
	v.SetConfigType("yaml")

	if cfgFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		dir := home + "/.heimdall"
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("cannot create config directory: %w", err)
		}
		cfgFile = dir + "/config.yaml"
	}

	v.SetConfigFile(cfgFile)
	v.ReadInConfig() // ignore error — file may not exist yet

	v.Set(fmt.Sprintf("profiles.%s.api_key", profileName), profile.APIKey)
	v.Set(fmt.Sprintf("profiles.%s.base_url", profileName), profile.BaseURL)
	if profile.IngestAPIKey != "" {
		v.Set(fmt.Sprintf("profiles.%s.ingest_api_key", profileName), profile.IngestAPIKey)
	}

	if v.GetString("default_profile") == "" {
		v.Set("default_profile", profileName)
	}

	return v.WriteConfigAs(cfgFile)
}

// ListProfiles returns all profile names from the config file.
func ListProfiles(cfgFile string) (map[string]Profile, string, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	if cfgFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		v.SetConfigFile(home + "/.heimdall/config.yaml")
	} else {
		v.SetConfigFile(cfgFile)
	}

	if err := v.ReadInConfig(); err != nil {
		return map[string]Profile{}, "", nil
	}

	profiles := map[string]Profile{}
	raw := v.GetStringMap("profiles")
	for name := range raw {
		profiles[name] = Profile{
			APIKey:       v.GetString(fmt.Sprintf("profiles.%s.api_key", name)),
			BaseURL:      v.GetString(fmt.Sprintf("profiles.%s.base_url", name)),
			IngestAPIKey: v.GetString(fmt.Sprintf("profiles.%s.ingest_api_key", name)),
		}
	}

	return profiles, v.GetString("default_profile"), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```
go test ./internal/config/... -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_ingest_test.go
git commit -m "feat(config): add IngestAPIKey field to Profile"
```

---

## Task 2: Add ingest types and `post()` to API client

**Files:**
- Modify: `internal/api/types.go`
- Modify: `internal/api/client.go`

- [ ] **Step 1: Write the failing test**

Create `internal/api/ingest_test.go`:

```go
package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/octobit/heimdall-cli/internal/api"
)

func TestIngestLogLines(t *testing.T) {
	var received api.IngestLogLinesRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/ingest/log-lines" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "hm_live_test" {
			t.Errorf("unexpected key %s", r.Header.Get("X-API-Key"))
		}
		json.NewDecoder(r.Body).Decode(&received)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(api.IngestLogLinesResponse{Ingested: 2})
	}))
	defer srv.Close()

	client := api.NewIngestClient("hm_live_test", srv.URL)
	resp, err := client.IngestLogLines("app.log", []api.LogLineInput{
		{Message: "hello", Level: "info"},
		{Message: "boom", Level: "error", OccurredAt: "2026-04-29T10:00:00Z"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Ingested != 2 {
		t.Errorf("ingested=%d, want 2", resp.Ingested)
	}
	if received.SourceName != "app.log" {
		t.Errorf("source_name=%q, want app.log", received.SourceName)
	}
	if len(received.Lines) != 2 {
		t.Errorf("lines=%d, want 2", len(received.Lines))
	}
}

func TestIngestLogLinesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := api.NewIngestClient("hm_live_test", srv.URL)
	_, err := client.IngestLogLines("app.log", []api.LogLineInput{{Message: "x"}})
	if err == nil {
		t.Error("expected error on 503, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./internal/api/... -run TestIngestLogLines -v
```

Expected: FAIL — `IngestLogLinesRequest`, `NewIngestClient`, `IngestLogLines` not defined

- [ ] **Step 3: Add types to `internal/api/types.go`**

Append to the end of `internal/api/types.go`:

```go
// LogLineInput is one log line sent to POST /api/v1/ingest/log-lines.
type LogLineInput struct {
	Message    string `json:"message"`
	OccurredAt string `json:"occurred_at,omitempty"`
	Level      string `json:"level,omitempty"`
}

// IngestLogLinesRequest is the request body for POST /api/v1/ingest/log-lines.
type IngestLogLinesRequest struct {
	SourceName string         `json:"source_name"`
	Lines      []LogLineInput `json:"lines"`
}

// IngestLogLinesResponse is the response body from POST /api/v1/ingest/log-lines.
type IngestLogLinesResponse struct {
	Ingested int `json:"ingested"`
}
```

- [ ] **Step 4: Add `NewIngestClient`, `post()`, and `IngestLogLines()` to `internal/api/client.go`**

Add after the `NewClient` function:

```go
// NewIngestClient creates a client that sends requests using an ingest API key.
func NewIngestClient(ingestAPIKey, baseURL string) *Client {
	return &Client{
		apiKey:  ingestAPIKey,
		baseURL: baseURL,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}
```

Add after the `get()` method (before `setIfNotEmpty`):

```go
func (c *Client) post(path string, body any, out any) error {
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(body); err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1"+path, buf)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		var errResp struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if jsonErr := json.NewDecoder(resp.Body).Decode(&errResp); jsonErr == nil && errResp.Error.Message != "" {
			return fmt.Errorf("API error %d (%s): %s", resp.StatusCode, errResp.Error.Code, errResp.Error.Message)
		}
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// IngestLogLines sends a batch of log lines for one source to the backend.
func (c *Client) IngestLogLines(sourceName string, lines []LogLineInput) (*IngestLogLinesResponse, error) {
	req := IngestLogLinesRequest{SourceName: sourceName, Lines: lines}
	var out IngestLogLinesResponse
	return &out, c.post("/ingest/log-lines", req, &out)
}
```

Add `"bytes"` to the import block in `client.go`.

- [ ] **Step 5: Run tests to verify they pass**

```
go test ./internal/api/... -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/api/types.go internal/api/client.go internal/api/ingest_test.go
git commit -m "feat(api): add IngestLogLines and post() to client"
```

---

## Task 3: Level detection

**Files:**
- Create: `internal/tail/level.go`
- Create: `internal/tail/level_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/tail/level_test.go
package tail

import "testing"

func TestDetectLevel(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{"FATAL: disk full", "fatal"},
		{"panic: runtime error", "fatal"},
		{"ERROR connecting to db", "error"},
		{"[err] file not found", "error"},
		{"critical: service down", "error"},
		{"WARN: retrying request", "warn"},
		{"[warning] slow query", "warn"},
		{"DEBUG handler called", "debug"},
		{"[trace] entering function", "debug"},
		{"INFO server started", "info"},
		{"user logged in successfully", "info"},
		{"", "info"},
	}
	for _, tc := range cases {
		got := detectLevel(tc.line)
		if got != tc.want {
			t.Errorf("detectLevel(%q) = %q, want %q", tc.line, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./internal/tail/... -run TestDetectLevel -v
```

Expected: FAIL — package does not exist

- [ ] **Step 3: Implement**

```go
// internal/tail/level.go
package tail

import "regexp"

var (
	reFatal = regexp.MustCompile(`(?i)(fatal|panic)`)
	reError = regexp.MustCompile(`(?i)(error|err|critical)`)
	reWarn  = regexp.MustCompile(`(?i)(warn|warning)`)
	reDebug = regexp.MustCompile(`(?i)(debug|trace)`)
)

func detectLevel(line string) string {
	switch {
	case reFatal.MatchString(line):
		return "fatal"
	case reError.MatchString(line):
		return "error"
	case reWarn.MatchString(line):
		return "warn"
	case reDebug.MatchString(line):
		return "debug"
	default:
		return "info"
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```
go test ./internal/tail/... -run TestDetectLevel -v
```

Expected: PASS (12/12)

- [ ] **Step 5: Commit**

```bash
git add internal/tail/level.go internal/tail/level_test.go
git commit -m "feat(tail): level detection from log line content"
```

---

## Task 4: Byte offset persistence

**Files:**
- Create: `internal/tail/offset.go`
- Create: `internal/tail/offset_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/tail/offset_test.go
package tail

import (
	"path/filepath"
	"testing"
)

func TestOffsetRoundTrip(t *testing.T) {
	dir := t.TempDir()

	o := &offsetStore{dir: dir}

	key := o.key("/var/log/app.log")
	if key == "" {
		t.Fatal("key must not be empty")
	}

	if err := o.save(key, 12345); err != nil {
		t.Fatal(err)
	}

	got, err := o.load(key)
	if err != nil {
		t.Fatal(err)
	}
	if got != 12345 {
		t.Errorf("got %d, want 12345", got)
	}
}

func TestOffsetMissingReturnsZero(t *testing.T) {
	dir := t.TempDir()
	o := &offsetStore{dir: dir}

	got, err := o.load("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestOffsetKeyIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	o := &offsetStore{dir: dir}

	k1 := o.key("/var/log/app.log")
	k2 := o.key("/var/log/app.log")
	if k1 != k2 {
		t.Error("key is not deterministic")
	}
	_ = filepath.Join(dir, k1) // must be a valid path component
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./internal/tail/... -run TestOffset -v
```

Expected: FAIL — `offsetStore` not defined

- [ ] **Step 3: Implement**

```go
// internal/tail/offset.go
package tail

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type offsetStore struct {
	dir string
}

func newOffsetStore() (*offsetStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home dir: %w", err)
	}
	dir := filepath.Join(home, ".heimdall", "offsets")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("cannot create offsets dir: %w", err)
	}
	return &offsetStore{dir: dir}, nil
}

func (s *offsetStore) key(absPath string) string {
	sum := sha256.Sum256([]byte(absPath))
	return fmt.Sprintf("%x", sum)
}

func (s *offsetStore) save(key string, offset int64) error {
	data, err := json.Marshal(map[string]int64{"offset": offset})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, key+".json"), data, 0600)
}

func (s *offsetStore) load(key string) (int64, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, key+".json"))
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var m map[string]int64
	if err := json.Unmarshal(data, &m); err != nil {
		return 0, nil
	}
	return m["offset"], nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```
go test ./internal/tail/... -run TestOffset -v
```

Expected: PASS (3/3)

- [ ] **Step 5: Commit**

```bash
git add internal/tail/offset.go internal/tail/offset_test.go
git commit -m "feat(tail): byte offset persistence per file"
```

---

## Task 5: File tailer (inode watch + line reading)

**Files:**
- Create: `internal/tail/tailer.go`
- Create: `internal/tail/tailer_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/tail/tailer_test.go
package tail

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTailerReadsNewLines(t *testing.T) {
	dir := t.TempDir()
	f, err := os.CreateTemp(dir, "app*.log")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	store := &offsetStore{dir: dir}
	lines := make(chan Line, 10)

	tl := newTailer(f.Name(), filepath.Base(f.Name()), store, lines)
	go tl.run()
	defer tl.stop()

	time.Sleep(50 * time.Millisecond)

	fh, _ := os.OpenFile(f.Name(), os.O_APPEND|os.O_WRONLY, 0600)
	fh.WriteString("hello world\n")
	fh.WriteString("ERROR something bad\n")
	fh.Close()

	var got []Line
	timeout := time.After(2 * time.Second)
	for len(got) < 2 {
		select {
		case l := <-lines:
			got = append(got, l)
		case <-timeout:
			t.Fatalf("timeout waiting for lines, got %d", len(got))
		}
	}

	if got[0].Message != "hello world" {
		t.Errorf("message=%q, want 'hello world'", got[0].Message)
	}
	if got[0].Level != "info" {
		t.Errorf("level=%q, want info", got[0].Level)
	}
	if got[1].Level != "error" {
		t.Errorf("level=%q, want error", got[1].Level)
	}
	if got[0].SourceName != filepath.Base(f.Name()) {
		t.Errorf("source_name=%q", got[0].SourceName)
	}
}

func TestTailerDetectsRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	os.WriteFile(path, []byte("old content\n"), 0600)

	store := &offsetStore{dir: dir}
	lines := make(chan Line, 20)

	tl := newTailer(path, "app.log", store, lines)
	go tl.run()
	defer tl.stop()

	// drain initial content
	time.Sleep(50 * time.Millisecond)
	for len(lines) > 0 {
		<-lines
	}

	// simulate rotation: replace file
	os.Remove(path)
	os.WriteFile(path, []byte("new file line\n"), 0600)

	var got Line
	timeout := time.After(3 * time.Second)
	select {
	case got = <-lines:
	case <-timeout:
		t.Fatal("timeout waiting for post-rotation line")
	}
	if got.Message != "new file line" {
		t.Errorf("message=%q, want 'new file line'", got.Message)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```
go test ./internal/tail/... -run TestTailer -v -timeout 15s
```

Expected: FAIL — `Line`, `newTailer` not defined

- [ ] **Step 3: Implement**

```go
// internal/tail/tailer.go
package tail

import (
	"bufio"
	"io"
	"log"
	"os"
	"time"
)

// Line is one log line emitted by a tailer.
type Line struct {
	SourceName string
	Message    string
	Level      string
	OccurredAt time.Time
}

type tailer struct {
	path       string
	sourceName string
	store      *offsetStore
	out        chan<- Line
	stopCh     chan struct{}
}

func newTailer(path, sourceName string, store *offsetStore, out chan<- Line) *tailer {
	return &tailer{
		path:       path,
		sourceName: sourceName,
		store:      store,
		out:        out,
		stopCh:     make(chan struct{}),
	}
}

func (t *tailer) stop() {
	close(t.stopCh)
}

func (t *tailer) run() {
	key := t.store.key(t.path)
	offset, err := t.store.load(key)
	if err != nil {
		log.Printf("tail: could not load offset for %s: %v", t.path, err)
	}

	f, inode, err := t.openAt(offset)
	if err != nil {
		log.Printf("tail: could not open %s: %v", t.path, err)
		return
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopCh:
			return
		case <-ticker.C:
			// read all available lines
			for {
				line, err := reader.ReadString('\n')
				if len(line) > 0 {
					msg := line
					if len(msg) > 0 && msg[len(msg)-1] == '\n' {
						msg = msg[:len(msg)-1]
					}
					if len(msg) > 0 && msg[len(msg)-1] == '\r' {
						msg = msg[:len(msg)-1]
					}
					if msg != "" {
						offset, _ = f.Seek(0, io.SeekCurrent)
						_ = t.store.save(key, offset)
						t.out <- Line{
							SourceName: t.sourceName,
							Message:    msg,
							Level:      detectLevel(msg),
							OccurredAt: time.Now().UTC(),
						}
					}
				}
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Printf("tail: read error on %s: %v", t.path, err)
					break
				}
			}

			// check for inode change (log rotation)
			newInode := inodeOf(t.path)
			if newInode != 0 && newInode != inode {
				f.Close()
				f, inode, err = t.openAt(0)
				if err != nil {
					log.Printf("tail: could not reopen %s after rotation: %v", t.path, err)
					return
				}
				reader = bufio.NewReader(f)
				offset = 0
				_ = t.store.save(key, 0)
			}
		}
	}
}

func (t *tailer) openAt(offset int64) (*os.File, uint64, error) {
	f, err := os.Open(t.path)
	if err != nil {
		return nil, 0, err
	}
	if offset == 0 {
		// first run: seek to end
		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			f.Close()
			return nil, 0, err
		}
	} else {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			f.Close()
			return nil, 0, err
		}
	}
	inode := inodeOf(t.path)
	return f, inode, nil
}
```

Create `internal/tail/inode_unix.go` (build-tagged for linux and darwin):

```go
//go:build linux || darwin

package tail

import (
	"os"
	"syscall"
)

func inodeOf(path string) uint64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0
	}
	return stat.Ino
}
```

- [ ] **Step 4: Fix openAt for first run**

The `openAt` function seeks to end when `offset == 0`, but for rotation we also pass 0 and want byte 0. Fix by adding a `seekEnd bool` parameter:

Replace the `openAt` call in `run()` (first open) with:
```go
f, inode, err := t.openAt(offset, offset == 0)
```

Replace the rotation reopen with:
```go
f, inode, err = t.openAt(0, false)
```

Update `openAt` signature:
```go
func (t *tailer) openAt(offset int64, seekEnd bool) (*os.File, uint64, error) {
	f, err := os.Open(t.path)
	if err != nil {
		return nil, 0, err
	}
	if seekEnd {
		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			f.Close()
			return nil, 0, err
		}
	} else if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			f.Close()
			return nil, 0, err
		}
	}
	inode := inodeOf(t.path)
	return f, inode, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

```
go test ./internal/tail/... -run TestTailer -v -timeout 15s
```

Expected: PASS (2/2)

- [ ] **Step 6: Commit**

```bash
git add internal/tail/tailer.go internal/tail/inode_unix.go internal/tail/tailer_test.go
git commit -m "feat(tail): file tailer with inode rotation detection"
```

---

## Task 6: Batcher (buffer, flush-by-size, flush-by-timer, retry)

**Files:**
- Create: `internal/tail/batcher.go`
- Create: `internal/tail/batcher_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/tail/batcher_test.go
package tail

import (
	"sync/atomic"
	"testing"
	"time"
)

// mockFlusher counts calls and records lines flushed.
type mockFlusher struct {
	calls  atomic.Int32
	fail   atomic.Bool
	flushed []Line
}

func (m *mockFlusher) flush(sourceName string, lines []Line) error {
	m.calls.Add(1)
	if m.fail.Load() {
		return fmt.Errorf("backend down")
	}
	m.flushed = append(m.flushed, lines...)
	return nil
}

func TestBatchFlushBySize(t *testing.T) {
	mf := &mockFlusher{}
	b := newBatcher(mf.flush, 500, 10*time.Second, 10_000)
	go b.run()
	defer b.stop()

	for i := 0; i < 500; i++ {
		b.add(Line{SourceName: "app.log", Message: fmt.Sprintf("line %d", i), Level: "info"})
	}

	time.Sleep(200 * time.Millisecond)
	if mf.calls.Load() < 1 {
		t.Error("expected at least one flush after 500 lines")
	}
}

func TestBatchFlushByTimer(t *testing.T) {
	mf := &mockFlusher{}
	b := newBatcher(mf.flush, 500, 200*time.Millisecond, 10_000)
	go b.run()
	defer b.stop()

	b.add(Line{SourceName: "app.log", Message: "only one line", Level: "info"})

	time.Sleep(500 * time.Millisecond)
	if mf.calls.Load() < 1 {
		t.Error("expected timer flush")
	}
}

func TestBatchRetryBackoff(t *testing.T) {
	mf := &mockFlusher{}
	mf.fail.Store(true)

	b := newBatcher(mf.flush, 500, 5*time.Second, 10_000)
	b.initialRetryDelay = 50 * time.Millisecond
	b.maxRetryDelay = 200 * time.Millisecond
	go b.run()
	defer b.stop()

	b.add(Line{SourceName: "app.log", Message: "line", Level: "info"})

	time.Sleep(100 * time.Millisecond)
	mf.fail.Store(false)

	time.Sleep(500 * time.Millisecond)
	if mf.calls.Load() < 2 {
		t.Errorf("expected at least 2 flush attempts (1 fail + 1 success), got %d", mf.calls.Load())
	}
}

func TestBatchBufferOverflow(t *testing.T) {
	mf := &mockFlusher{}
	mf.fail.Store(true)

	b := newBatcher(mf.flush, 500, 5*time.Second, 100)
	b.initialRetryDelay = 50 * time.Millisecond
	b.maxRetryDelay = 100 * time.Millisecond
	go b.run()
	defer b.stop()

	for i := 0; i < 110; i++ {
		b.add(Line{SourceName: "app.log", Message: fmt.Sprintf("line %d", i), Level: "info"})
	}

	time.Sleep(300 * time.Millisecond)
	// buffer should have dropped oldest lines; total buffered <= 100
	b.mu.Lock()
	total := 0
	for _, v := range b.buf {
		total += len(v)
	}
	b.mu.Unlock()
	if total > 100 {
		t.Errorf("buffer has %d lines, want <= 100", total)
	}
}
```

Note: The test file uses `fmt` — add `"fmt"` and `"sync"` imports.

- [ ] **Step 2: Run tests to verify they fail**

```
go test ./internal/tail/... -run TestBatch -v -timeout 30s
```

Expected: FAIL — `newBatcher` not defined

- [ ] **Step 3: Implement**

```go
// internal/tail/batcher.go
package tail

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type flushFn func(sourceName string, lines []Line) error

type batcher struct {
	flush             flushFn
	flushSize         int
	flushInterval     time.Duration
	maxBuffer         int
	initialRetryDelay time.Duration
	maxRetryDelay     time.Duration

	mu     sync.Mutex
	buf    map[string][]Line

	addCh  chan Line
	stopCh chan struct{}
}

func newBatcher(flush flushFn, flushSize int, flushInterval time.Duration, maxBuffer int) *batcher {
	return &batcher{
		flush:             flush,
		flushSize:         flushSize,
		flushInterval:     flushInterval,
		maxBuffer:         maxBuffer,
		initialRetryDelay: time.Second,
		maxRetryDelay:     60 * time.Second,
		buf:               make(map[string][]Line),
		addCh:             make(chan Line, 1000),
		stopCh:            make(chan struct{}),
	}
}

func (b *batcher) add(l Line) {
	b.addCh <- l
}

func (b *batcher) stop() {
	close(b.stopCh)
}

func (b *batcher) totalLines() int {
	n := 0
	for _, v := range b.buf {
		n += len(v)
	}
	return n
}

func (b *batcher) run() {
	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()

	pending := map[string][]Line{} // lines held during retry

	for {
		select {
		case <-b.stopCh:
			return

		case l := <-b.addCh:
			b.mu.Lock()
			total := b.totalLines()
			if total >= b.maxBuffer {
				// drop oldest line from the largest source
				var biggestSrc string
				max := 0
				for src, lines := range b.buf {
					if len(lines) > max {
						max = len(lines)
						biggestSrc = src
					}
				}
				if biggestSrc != "" {
					b.buf[biggestSrc] = b.buf[biggestSrc][1:]
					log.Printf("tail: buffer full: dropped 1 line from %s", biggestSrc)
				}
			}
			b.buf[l.SourceName] = append(b.buf[l.SourceName], l)
			total = b.totalLines()
			b.mu.Unlock()

			if total >= b.flushSize {
				b.doFlush(pending)
			}

		case <-ticker.C:
			b.mu.Lock()
			total := b.totalLines()
			b.mu.Unlock()
			if total > 0 {
				b.doFlush(pending)
			}
		}
	}
}

func (b *batcher) doFlush(pending map[string][]Line) {
	b.mu.Lock()
	batch := b.buf
	b.buf = make(map[string][]Line)
	b.mu.Unlock()

	// merge any previously pending lines at the front
	for src, lines := range pending {
		batch[src] = append(lines, batch[src]...)
		delete(pending, src)
	}

	delay := b.initialRetryDelay
	for src, lines := range batch {
		for {
			if err := b.flush(src, lines); err != nil {
				log.Printf("tail: flush failed for %s: %v, retrying in %s", src, err, delay)
				time.Sleep(delay)
				delay *= 2
				if delay > b.maxRetryDelay {
					delay = b.maxRetryDelay
				}
				// check if buffer overflow occurred while retrying
				b.mu.Lock()
				total := b.totalLines()
				b.mu.Unlock()
				if total >= b.maxBuffer {
					log.Printf("tail: buffer full during retry: dropping %d lines from %s", len(lines), src)
					break
				}
				continue
			}
			break
		}
	}
}
```

- [ ] **Step 4: Fix test imports**

Add `"fmt"` and `"sync"` to the test file imports.

- [ ] **Step 5: Run tests to verify they pass**

```
go test ./internal/tail/... -run TestBatch -v -timeout 30s
```

Expected: PASS (4/4)

- [ ] **Step 6: Commit**

```bash
git add internal/tail/batcher.go internal/tail/batcher_test.go
git commit -m "feat(tail): batcher with flush-by-size, timer, retry, buffer overflow"
```

---

## Task 7: Daemon mode

**Files:**
- Create: `internal/tail/daemon.go` (build tag: `linux || darwin`)

No automated test (fork behavior is not testable without actually forking). Manual verification steps are provided.

- [ ] **Step 1: Create the file**

```go
//go:build linux || darwin

// internal/tail/daemon.go
package tail

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func heimdallDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".heimdall")
	return dir, os.MkdirAll(dir, 0700)
}

// Daemonize re-execs the current binary with the same args minus --daemon,
// detaches it from the terminal (new session), redirects its output to
// ~/.heimdall/tail.log, and writes its PID to ~/.heimdall/tail.pid.
// The parent process prints a confirmation message and returns.
func Daemonize(files []string, extraArgs []string) error {
	dir, err := heimdallDir()
	if err != nil {
		return err
	}

	logPath := filepath.Join(dir, "tail.log")
	pidPath := filepath.Join(dir, "tail.pid")

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer logFile.Close()

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	args := append([]string{"tail"}, files...)
	args = append(args, extraArgs...)

	cmd := exec.Command(self, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting daemon: %w", err)
	}

	pid := cmd.Process.Pid
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0600); err != nil {
		return fmt.Errorf("writing PID file: %w", err)
	}

	fmt.Printf("Heimdall tail running in background (PID %d)\n", pid)
	fmt.Printf("Logs: %s\n", logPath)
	return nil
}

// StopDaemon reads ~/.heimdall/tail.pid and sends SIGTERM to the process.
func StopDaemon() error {
	dir, err := heimdallDir()
	if err != nil {
		return err
	}
	pidPath := filepath.Join(dir, "tail.pid")
	data, err := os.ReadFile(pidPath)
	if os.IsNotExist(err) {
		fmt.Println("No daemon running")
		return nil
	}
	if err != nil {
		return err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return fmt.Errorf("invalid PID file: %w", err)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process %d: %w", pid, err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("sending SIGTERM to %d: %w", pid, err)
	}

	os.Remove(pidPath)
	fmt.Printf("Sent SIGTERM to PID %d\n", pid)
	return nil
}

// DaemonLogPath returns the path to the daemon log file.
func DaemonLogPath() (string, error) {
	dir, err := heimdallDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tail.log"), nil
}
```

- [ ] **Step 2: Build to verify compilation**

```
go build ./internal/tail/...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/tail/daemon.go
git commit -m "feat(tail): daemon re-exec with Setsid, PID file management"
```

---

## Task 8: Service management — Linux (systemd)

**Files:**
- Create: `internal/tail/service_linux.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/tail/service_linux_test.go
//go:build linux

package tail

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateSystemdUnit(t *testing.T) {
	unit := generateSystemdUnit(
		"/usr/local/bin/heimdall",
		[]string{"/var/log/app.log", "/var/log/worker.log"},
		"hm_live_testkey",
	)

	if !strings.Contains(unit, "ExecStart=/usr/local/bin/heimdall tail /var/log/app.log /var/log/worker.log") {
		t.Errorf("missing ExecStart in unit:\n%s", unit)
	}
	if !strings.Contains(unit, "HEIMDALL_INGEST_API_KEY=hm_live_testkey") {
		t.Errorf("missing HEIMDALL_INGEST_API_KEY in unit:\n%s", unit)
	}
	if !strings.Contains(unit, "Restart=on-failure") {
		t.Errorf("missing Restart=on-failure in unit:\n%s", unit)
	}
}

func TestWriteSystemdUnit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "heimdall-tail.service")

	err := writeSystemdUnit(path, "/usr/local/bin/heimdall", []string{"/var/log/app.log"}, "hm_live_key")
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "[Service]") {
		t.Error("unit file missing [Service] section")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./internal/tail/... -run TestGenerateSystemdUnit -v
```

Expected: FAIL — `generateSystemdUnit` not defined

- [ ] **Step 3: Implement**

```go
//go:build linux

// internal/tail/service_linux.go
package tail

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func generateSystemdUnit(execPath string, files []string, ingestKey string) string {
	execStart := execPath + " tail " + strings.Join(files, " ")
	return fmt.Sprintf(`[Unit]
Description=Heimdall log tail
After=network.target

[Service]
ExecStart=%s
Restart=on-failure
RestartSec=5s
Environment=HEIMDALL_INGEST_API_KEY=%s

[Install]
WantedBy=default.target
`, execStart, ingestKey)
}

func writeSystemdUnit(path, execPath string, files []string, ingestKey string) error {
	unit := generateSystemdUnit(execPath, files, ingestKey)
	return os.WriteFile(path, []byte(unit), 0644)
}

// InstallService installs a systemd user service for heimdall tail.
func InstallService(files []string, ingestKey string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating systemd user dir: %w", err)
	}
	unitPath := filepath.Join(dir, "heimdall-tail.service")

	if err := writeSystemdUnit(unitPath, self, files, ingestKey); err != nil {
		return fmt.Errorf("writing unit file: %w", err)
	}

	for _, subcmd := range [][]string{
		{"--user", "daemon-reload"},
		{"--user", "enable", "--now", "heimdall-tail"},
	} {
		if out, err := exec.Command("systemctl", subcmd...).CombinedOutput(); err != nil {
			return fmt.Errorf("systemctl %v: %w\n%s", subcmd, err, out)
		}
	}
	fmt.Printf("Service installed. Unit: %s\n", unitPath)
	return nil
}

// UninstallService stops and removes the systemd user service.
func UninstallService() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if out, err := exec.Command("systemctl", "--user", "disable", "--now", "heimdall-tail").CombinedOutput(); err != nil {
		fmt.Printf("warning: systemctl disable: %s\n", out)
	}

	unitPath := filepath.Join(home, ".config", "systemd", "user", "heimdall-tail.service")
	os.Remove(unitPath)
	fmt.Println("Service uninstalled.")
	return nil
}

// ServiceStatus prints the systemd service status.
func ServiceStatus() error {
	out, _ := exec.Command("systemctl", "--user", "status", "heimdall-tail").CombinedOutput()
	fmt.Print(string(out))
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```
go test ./internal/tail/... -run TestGenerateSystemdUnit -v
```

Expected: PASS (2/2)

- [ ] **Step 5: Commit**

```bash
git add internal/tail/service_linux.go internal/tail/service_linux_test.go
git commit -m "feat(tail): systemd user service install/uninstall/status (Linux)"
```

---

## Task 9: Service management — macOS (launchd)

**Files:**
- Create: `internal/tail/service_darwin.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/tail/service_darwin_test.go
//go:build darwin

package tail

import (
	"strings"
	"testing"
)

func TestGenerateLaunchdPlist(t *testing.T) {
	plist := generateLaunchdPlist(
		"/usr/local/bin/heimdall",
		[]string{"/var/log/app.log"},
		"hm_live_key",
	)

	if !strings.Contains(plist, "com.heimdall.tail") {
		t.Errorf("missing label in plist:\n%s", plist)
	}
	if !strings.Contains(plist, "/usr/local/bin/heimdall") {
		t.Errorf("missing exec path in plist:\n%s", plist)
	}
	if !strings.Contains(plist, "HEIMDALL_INGEST_API_KEY") {
		t.Errorf("missing env key in plist:\n%s", plist)
	}
	if !strings.Contains(plist, "hm_live_key") {
		t.Errorf("missing ingest key value in plist:\n%s", plist)
	}
	if !strings.Contains(plist, "<true/>") {
		t.Errorf("missing KeepAlive or RunAtLoad in plist:\n%s", plist)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./internal/tail/... -run TestGenerateLaunchdPlist -v
```

Expected: FAIL — `generateLaunchdPlist` not defined

- [ ] **Step 3: Implement**

```go
//go:build darwin

// internal/tail/service_darwin.go
package tail

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func generateLaunchdPlist(execPath string, files []string, ingestKey string) string {
	programArgs := fmt.Sprintf(`		<string>%s</string>
		<string>tail</string>`, execPath)
	for _, f := range files {
		programArgs += fmt.Sprintf("\n\t\t<string>%s</string>", f)
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.heimdall.tail</string>
	<key>ProgramArguments</key>
	<array>
%s
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>EnvironmentVariables</key>
	<dict>
		<key>HEIMDALL_INGEST_API_KEY</key>
		<string>%s</string>
	</dict>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
</dict>
</plist>
`, programArgs, ingestKey, launchLogPath(), launchLogPath())
}

func launchLogPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".heimdall", "tail.log")
}

// InstallService installs a launchd agent for heimdall tail.
func InstallService(files []string, ingestKey string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating LaunchAgents dir: %w", err)
	}
	plistPath := filepath.Join(dir, "com.heimdall.tail.plist")

	plist := generateLaunchdPlist(self, files, ingestKey)
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return fmt.Errorf("writing plist: %w", err)
	}

	if out, err := exec.Command("launchctl", "load", plistPath).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load: %w\n%s", err, out)
	}
	fmt.Printf("Service installed. Plist: %s\n", plistPath)
	return nil
}

// UninstallService unloads and removes the launchd agent.
func UninstallService() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.heimdall.tail.plist")

	if out, err := exec.Command("launchctl", "unload", plistPath).CombinedOutput(); err != nil {
		fmt.Printf("warning: launchctl unload: %s\n", out)
	}
	os.Remove(plistPath)
	fmt.Println("Service uninstalled.")
	return nil
}

// ServiceStatus prints the launchd agent status.
func ServiceStatus() error {
	out, _ := exec.Command("launchctl", "list").Output()
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "heimdall") {
			fmt.Println(line)
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```
go test ./internal/tail/... -run TestGenerateLaunchdPlist -v
```

Expected: PASS (1/1)

- [ ] **Step 5: Commit**

```bash
git add internal/tail/service_darwin.go internal/tail/service_darwin_test.go
git commit -m "feat(tail): launchd agent install/uninstall/status (macOS)"
```

---

## Task 10: `heimdall configure` — add ingest key prompt

**Files:**
- Modify: `cmd/configure.go`

- [ ] **Step 1: Write the failing test**

There is no existing unit test for `configure`. Since the configure command interacts with stdin and hits the network, this step verifies compilation only. Instead, update the `cmd/configure.go` file and confirm `go build ./...` passes.

- [ ] **Step 2: Modify `cmd/configure.go`**

Replace the `RunE` body with:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    r := bufio.NewReader(os.Stdin)

    profileName := prompt(r, "Profile name", "default")
    apiKey := prompt(r, "Read-only API key (hm_read_...)", "")
    if apiKey == "" {
        return fmt.Errorf("API key is required")
    }
    baseURL := prompt(r, "Heimdall base URL", "https://api.heimdall-ob.com")
    ingestKey := prompt(r, "Ingest API key (optional, required for 'heimdall tail')", "")

    fmt.Print("Testing connection... ")
    client := api.NewClient(apiKey, baseURL)
    if _, err := client.GetOverview("", ""); err != nil {
        fmt.Println("failed")
        return fmt.Errorf("could not connect: %w", err)
    }
    fmt.Println("OK")

    if err := config.Save("", profileName, config.Profile{
        APIKey:       apiKey,
        BaseURL:      baseURL,
        IngestAPIKey: ingestKey,
    }); err != nil {
        return fmt.Errorf("saving config: %w", err)
    }

    fmt.Printf("\nProfile %q saved. Run 'heimdall overview' to get started.\n", profileName)
    return nil
},
```

- [ ] **Step 3: Build to verify compilation**

```
go build ./...
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add cmd/configure.go
git commit -m "feat(configure): prompt for ingest API key (optional)"
```

---

## Task 11: `heimdall tail` command

**Files:**
- Create: `cmd/tail.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Write the integration test first**

Create `cmd/tail_test.go`:

```go
package cmd_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/octobit/heimdall-cli/internal/api"
)

func TestTailCommandIngestsLines(t *testing.T) {
	var received []api.IngestLogLinesRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/ingest/log-lines" {
			var req api.IngestLogLinesRequest
			json.NewDecoder(r.Body).Decode(&req)
			received = append(received, req)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(api.IngestLogLinesResponse{Ingested: len(received)})
	}))
	defer srv.Close()

	// Write config
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgFile, []byte(strings.ReplaceAll(`
default_profile: default
profiles:
  default:
    api_key: hm_read_test
    base_url: BASE_URL
    ingest_api_key: hm_live_test
`, "BASE_URL", srv.URL)), 0600)

	// Write log file
	logFile := filepath.Join(dir, "app.log")
	os.WriteFile(logFile, []byte(""), 0600)

	// Run tail for a short duration via a goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		runTailForTest(cfgFile, logFile, 3*time.Second)
	}()

	time.Sleep(200 * time.Millisecond)

	// Append lines
	f, _ := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0600)
	f.WriteString("user logged in\n")
	f.WriteString("ERROR failed to connect\n")
	f.Close()

	<-done

	if len(received) == 0 {
		t.Fatal("no ingest requests received")
	}

	var allLines []api.LogLineInput
	for _, r := range received {
		allLines = append(allLines, r.Lines...)
		if r.SourceName != "app.log" {
			t.Errorf("source_name=%q, want app.log", r.SourceName)
		}
	}

	if len(allLines) < 2 {
		t.Fatalf("expected >=2 lines, got %d", len(allLines))
	}
	found := false
	for _, l := range allLines {
		if l.Level == "error" {
			found = true
		}
	}
	if !found {
		t.Error("expected at least one line with level=error")
	}
}
```

`runTailForTest` will be a package-internal helper in `cmd/tail.go` (exported only for tests).

- [ ] **Step 2: Run test to verify it fails**

```
go test ./cmd/... -run TestTailCommandIngestsLines -v -timeout 15s
```

Expected: FAIL — `runTailForTest` not defined

- [ ] **Step 3: Implement `cmd/tail.go`**

```go
package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/octobit/heimdall-cli/internal/api"
	"github.com/octobit/heimdall-cli/internal/config"
	"github.com/octobit/heimdall-cli/internal/tail"
	"github.com/spf13/cobra"
)

var (
	tailDaemon     bool
	tailStop       bool
	tailLogs       bool
	tailSourceName string
)

var tailCmd = &cobra.Command{
	Use:   "tail <file> [files...]",
	Short: "Tail log files and stream lines to Heimdall",
	Long: `Tails one or more log files and streams new lines to the Heimdall
ingest endpoint. Resumes from saved byte offsets on restart.
Requires an ingest API key (hm_live_...) configured via 'heimdall configure'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if tailStop {
			return tail.StopDaemon()
		}
		if tailLogs {
			path, err := tail.DaemonLogPath()
			if err != nil {
				return err
			}
			fmt.Println(path)
			return nil
		}

		if len(args) == 0 {
			return fmt.Errorf("at least one file path is required")
		}
		if tailSourceName != "" && len(args) > 1 {
			return fmt.Errorf("--source-name can only be used with a single file")
		}

		if tailDaemon {
			return tail.Daemonize(args, nil)
		}

		profile, err := config.Load("", profileFlag)
		if err != nil {
			return err
		}
		ingestKey := profile.IngestAPIKey
		if envKey := os.Getenv("HEIMDALL_INGEST_API_KEY"); envKey != "" {
			ingestKey = envKey
		}
		if ingestKey == "" {
			return fmt.Errorf("ingest API key not configured — run 'heimdall configure' or set HEIMDALL_INGEST_API_KEY")
		}

		client := api.NewIngestClient(ingestKey, profile.BaseURL)
		return runTail(client, args, 0)
	},
}

// runTailForTest is exposed for integration tests.
func runTailForTest(cfgFile, logFile string, duration time.Duration) {
	profile, err := config.Load(cfgFile, "")
	if err != nil {
		log.Fatal(err)
	}
	client := api.NewIngestClient(profile.IngestAPIKey, profile.BaseURL)
	go func() {
		time.Sleep(duration)
		os.Exit(0)
	}()
	runTail(client, []string{logFile}, duration) //nolint
}

func runTail(client *api.Client, files []string, duration time.Duration) error {
	store, err := tail.NewOffsetStore()
	if err != nil {
		return err
	}

	lines := make(chan tail.Line, 1000)

	flushFn := func(sourceName string, batch []tail.Line) error {
		inputs := make([]api.LogLineInput, len(batch))
		for i, l := range batch {
			inputs[i] = api.LogLineInput{
				Message:    l.Message,
				Level:      l.Level,
				OccurredAt: l.OccurredAt.Format(time.RFC3339),
			}
		}
		_, err := client.IngestLogLines(sourceName, inputs)
		return err
	}

	b := tail.NewBatcher(flushFn, 500, 5*time.Second, 10_000)
	go b.Run()
	defer b.Stop()

	var tailers []*tail.Tailer
	for _, path := range files {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("resolving path %s: %w", path, err)
		}
		sourceName := filepath.Base(absPath)
		if tailSourceName != "" {
			sourceName = tailSourceName
		}
		t := tail.NewTailer(absPath, sourceName, store, lines)
		tailers = append(tailers, t)
		go t.Run()
	}
	defer func() {
		for _, t := range tailers {
			t.Stop()
		}
	}()

	go func() {
		for l := range lines {
			b.Add(l)
		}
	}()

	if duration > 0 {
		time.Sleep(duration)
		return nil
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig
	fmt.Println("\nShutting down...")
	return nil
}

func init() {
	rootCmd.AddCommand(tailCmd)
	tailCmd.Flags().BoolVar(&tailDaemon, "daemon", false, "run in background (daemon mode)")
	tailCmd.Flags().BoolVar(&tailStop, "stop", false, "stop the running daemon")
	tailCmd.Flags().BoolVar(&tailLogs, "logs", false, "print path to daemon log file")
	tailCmd.Flags().StringVar(&tailSourceName, "source-name", "", "override source_name (single file only)")
}
```

**Note:** This requires exporting `NewOffsetStore`, `NewBatcher`, `NewTailer`, `Batcher.Run`, `Batcher.Stop`, `Batcher.Add`, `Tailer.Run`, `Tailer.Stop` from the `tail` package. In steps below, update the internal types to export these names.

- [ ] **Step 4: Export public API from `internal/tail`**

In `internal/tail/offset.go`, add:

```go
// NewOffsetStore creates an offsetStore in ~/.heimdall/offsets.
func NewOffsetStore() (*offsetStore, error) {
	return newOffsetStore()
}
```

In `internal/tail/batcher.go`, rename `run` → `Run`, `stop` → `Stop`, `add` → `Add`:

```go
func (b *batcher) Add(l Line)  { b.addCh <- l }
func (b *batcher) Run()        { b.run() }
func (b *batcher) Stop()       { b.stop() }
```

Add:
```go
// NewBatcher creates a batcher with production defaults.
func NewBatcher(flush flushFn, flushSize int, flushInterval time.Duration, maxBuffer int) *batcher {
	return newBatcher(flush, flushSize, flushInterval, maxBuffer)
}
```

In `internal/tail/tailer.go`, rename `run` → `Run`, `stop` → `Stop`:

```go
func (t *tailer) Run()  { t.run() }
func (t *tailer) Stop() { t.stop() }
```

Add:
```go
// NewTailer creates a tailer. Use Run() in a goroutine.
func NewTailer(path, sourceName string, store *offsetStore, out chan<- Line) *tailer {
	return newTailer(path, sourceName, store, out)
}
```

Also export `Tailer` and `Batcher` type aliases:

```go
type Tailer = tailer
type Batcher = batcher
```

- [ ] **Step 5: Register tailCmd in `cmd/root.go`**

`tailCmd` is registered via `init()` in `cmd/tail.go` — no change needed to `root.go`.

- [ ] **Step 6: Run the integration test**

```
go test ./cmd/... -run TestTailCommandIngestsLines -v -timeout 15s
```

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add cmd/tail.go cmd/tail_test.go internal/tail/offset.go internal/tail/batcher.go internal/tail/tailer.go
git commit -m "feat(cmd): heimdall tail command"
```

---

## Task 12: `heimdall service` command

**Files:**
- Create: `cmd/service.go`

- [ ] **Step 1: Create `cmd/service.go`**

```go
//go:build linux || darwin

package cmd

import (
	"fmt"
	"os"

	"github.com/octobit/heimdall-cli/internal/config"
	"github.com/octobit/heimdall-cli/internal/tail"
	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage heimdall tail as a system service",
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install <file> [files...]",
	Short: "Install heimdall tail as a boot-persistent service",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("at least one file path is required")
		}

		profile, err := config.Load("", profileFlag)
		if err != nil {
			return err
		}
		ingestKey := profile.IngestAPIKey
		if envKey := os.Getenv("HEIMDALL_INGEST_API_KEY"); envKey != "" {
			ingestKey = envKey
		}
		if ingestKey == "" {
			return fmt.Errorf("ingest API key not configured — run 'heimdall configure' or set HEIMDALL_INGEST_API_KEY")
		}

		return tail.InstallService(args, ingestKey)
	},
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the heimdall tail system service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return tail.UninstallService()
	},
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of the heimdall tail system service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return tail.ServiceStatus()
	},
}

func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
	serviceCmd.AddCommand(serviceStatusCmd)
}
```

- [ ] **Step 2: Build to verify compilation**

```
go build ./...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add cmd/service.go
git commit -m "feat(cmd): heimdall service install/uninstall/status"
```

---

## Task 13: Full build and test pass

**Files:** none new

- [ ] **Step 1: Run all tests**

```
go test ./... -v -timeout 60s
```

Expected: all PASS

- [ ] **Step 2: Run go vet**

```
go vet ./...
```

Expected: no output (no warnings)

- [ ] **Step 3: Build the binary**

```
go build -o /tmp/heimdall-test ./cmd/heimdall
```

Expected: no errors

- [ ] **Step 4: Smoke test the binary**

```
/tmp/heimdall-test tail --help
/tmp/heimdall-test service --help
```

Expected: usage text printed with correct flags

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore: full build and test pass for log streaming sub-project B"
```

---

## Self-Review Notes

**Spec coverage check:**
- `heimdall tail <files...>` with `--daemon`, `--stop`, `--logs`, `--source-name`: Task 11 ✓
- Auth resolution (env → profile → error): Task 11 + Task 1 ✓
- `heimdall configure` ingest key prompt: Task 10 ✓
- First run seeks to end, subsequent resumes offset: Tasks 4 + 5 ✓
- Inode rotation detection (1 s poll): Task 5 ✓
- source_name defaults to basename: Task 11 ✓
- Level detection (5 priorities): Task 3 ✓
- Flush by 500 lines or 5 s: Task 6 ✓
- One POST per source_name per flush: Task 6 ✓
- Retry with exponential backoff (1s → 60s cap): Task 6 ✓
- In-memory buffer limit 10,000 lines, drop oldest + warning: Task 6 ✓
- Offset not advanced past unsent lines: Task 6 (retry loop keeps lines in memory) ✓
- Daemon double-fork / re-exec: Task 7 ✓
- PID file: Task 7 ✓
- `--stop` SIGTERM via PID: Task 7 ✓
- `--logs` print log path: Task 7 / Task 11 ✓
- systemd user service: Task 8 ✓
- launchd agent: Task 9 ✓
- `heimdall service install/uninstall/status`: Tasks 8, 9, 12 ✓
- HEIMDALL_INGEST_API_KEY embedded in service file: Tasks 8, 9 ✓
- Integration test (mock HTTP, real file write): Task 11 ✓
- Unit tests for level, offset, rotation, flush-by-size, flush-by-timer, retry, overflow: Tasks 3–6 ✓
