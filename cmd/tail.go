//go:build linux || darwin

package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
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
Requires an ingest API key (hm_live_...) — configure with 'heimdall configure'.`,
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
		return runTail(client, args, tailSourceName, 0)
	},
}

// runTailWithConfig is used by integration tests to run the tailer with a specific config file.
func runTailWithConfig(cfgFile, logFile string, dur time.Duration) {
	profile, err := config.Load(cfgFile, "")
	if err != nil {
		return
	}
	client := api.NewIngestClient(profile.IngestAPIKey, profile.BaseURL)
	runTail(client, []string{logFile}, "", dur) //nolint:errcheck
}

func runTail(client *api.Client, files []string, sourceName string, dur time.Duration) error {
	store, err := tail.NewOffsetStore()
	if err != nil {
		return err
	}

	lines := make(chan tail.Line, 1000)

	flushFn := func(src string, batch []tail.Line) error {
		inputs := make([]api.LogLineInput, len(batch))
		for i, l := range batch {
			inputs[i] = api.LogLineInput{
				Message:    l.Message,
				Level:      l.Level,
				OccurredAt: l.OccurredAt.Format(time.RFC3339),
			}
		}
		_, err := client.IngestLogLines(src, inputs)
		return err
	}

	b := tail.NewBatcher(flushFn, 500, 1*time.Second, 10_000)
	var batcherWg sync.WaitGroup
	batcherWg.Add(1)
	go func() {
		defer batcherWg.Done()
		b.Run()
	}()

	var tailerWg sync.WaitGroup
	var tailers []*tail.Tailer
	for _, path := range files {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("resolving path %s: %w", path, err)
		}
		src := filepath.Base(absPath)
		if sourceName != "" {
			src = sourceName
		}
		t := tail.NewTailer(absPath, src, store, lines)
		tailers = append(tailers, t)
		tailerWg.Add(1)
		go func(t *tail.Tailer) {
			defer tailerWg.Done()
			t.Run()
		}(t)
	}

	var forwardWg sync.WaitGroup
	forwardWg.Add(1)
	go func() {
		defer forwardWg.Done()
		for l := range lines {
			b.Add(l)
		}
	}()

	// Shutdown: stop tailers → wait for tailers → close lines → drain forwarder → stop batcher
	defer func() {
		for _, t := range tailers {
			t.Stop()
		}
		tailerWg.Wait()  // wait for all tailers to finish writing to lines
		close(lines)     // signal forwarder to drain and exit
		forwardWg.Wait() // wait for forwarder to finish calling b.Add
		b.Stop()
		batcherWg.Wait()
	}()

	if dur > 0 {
		time.Sleep(dur)
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
	tailCmd.Flags().BoolVar(&tailDaemon, "daemon", false, "run in background")
	tailCmd.Flags().BoolVar(&tailStop, "stop", false, "stop the running daemon")
	tailCmd.Flags().BoolVar(&tailLogs, "logs", false, "print path to daemon log file")
	tailCmd.Flags().StringVar(&tailSourceName, "source-name", "", "override source_name (single file only)")
}
