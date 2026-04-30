package tail

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type mockFlusher struct {
	calls   atomic.Int32
	fail    atomic.Bool
	mu      sync.Mutex
	flushed []Line
}

func (m *mockFlusher) flush(sourceName string, lines []Line) error {
	m.calls.Add(1)
	if m.fail.Load() {
		return fmt.Errorf("backend down")
	}
	m.mu.Lock()
	m.flushed = append(m.flushed, lines...)
	m.mu.Unlock()
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

	b := newBatcher(mf.flush, 1, 5*time.Second, 10_000)
	b.initialRetryDelay = 50 * time.Millisecond
	b.maxRetryDelay = 200 * time.Millisecond
	go b.run()
	defer b.stop()

	b.add(Line{SourceName: "app.log", Message: "line", Level: "info"})

	time.Sleep(100 * time.Millisecond)
	mf.fail.Store(false)

	time.Sleep(500 * time.Millisecond)
	if mf.calls.Load() < 2 {
		t.Errorf("expected >=2 flush attempts (fail+success), got %d", mf.calls.Load())
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
