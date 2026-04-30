package tail

import (
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

	mu       sync.Mutex
	buf      map[string][]Line
	stopOnce sync.Once

	addCh   chan Line
	stopCh  chan struct{}
	flushCh chan struct{} // signals the flusher goroutine to flush now
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
		flushCh:           make(chan struct{}, 1),
	}
}

func (b *batcher) totalLines() int {
	n := 0
	for _, v := range b.buf {
		n += len(v)
	}
	return n
}

// signalFlush sends a non-blocking signal to the flusher goroutine.
func (b *batcher) signalFlush() {
	select {
	case b.flushCh <- struct{}{}:
	default:
	}
}

func (b *batcher) add(l Line) {
	b.addCh <- l
}

func (b *batcher) stop() {
	b.stopOnce.Do(func() { close(b.stopCh) })
}

// run is the main loop: ingests lines and signals flushes by size or timer.
// A separate flusher goroutine handles the actual flush + retry.
func (b *batcher) run() {
	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()

	// Start the flusher goroutine.
	done := make(chan struct{})
	go func() {
		defer close(done)
		b.flusher()
	}()

	for {
		select {
		case <-b.stopCh:
			// Signal flusher to stop and wait for it.
			b.signalFlush()
			<-done
			return

		case l := <-b.addCh:
			b.mu.Lock()
			total := b.totalLines()
			if total >= b.maxBuffer {
				// Drop oldest line from the largest source.
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
			if b.totalLines() >= b.flushSize {
				b.signalFlush()
			}
			b.mu.Unlock()

		case <-ticker.C:
			b.mu.Lock()
			total := b.totalLines()
			b.mu.Unlock()
			if total > 0 {
				b.signalFlush()
			}
		}
	}
}

// flusher runs in a goroutine, waiting for flush signals and executing retried flushes.
func (b *batcher) flusher() {
	for {
		select {
		case <-b.stopCh:
			// drain any pending flush signal so we don't drop lines buffered just before shutdown
			select {
			case <-b.flushCh:
				b.doFlush()
			default:
			}
			return
		case <-b.flushCh:
			b.doFlush()
		}
	}
}

func (b *batcher) doFlush() {
	b.mu.Lock()
	if b.totalLines() == 0 {
		b.mu.Unlock()
		return
	}
	batch := b.buf
	b.buf = make(map[string][]Line)
	b.mu.Unlock()

	for src, lines := range batch {
		delay := b.initialRetryDelay
		for {
			if err := b.flush(src, lines); err != nil {
				log.Printf("tail: flush failed for %s: %v, retrying in %s", src, err, delay)
				select {
				case <-b.stopCh:
					return
				case <-time.After(delay):
				}
				delay *= 2
				if delay > b.maxRetryDelay {
					delay = b.maxRetryDelay
				}
				continue
			}
			break
		}
	}
}

// Add, Run, Stop are the exported API for cmd/tail.go.
func (b *batcher) Add(l Line) { b.add(l) }
func (b *batcher) Run()       { b.run() }
func (b *batcher) Stop()      { b.stop() }

// NewBatcher creates a batcher with production defaults.
func NewBatcher(flush flushFn, flushSize int, flushInterval time.Duration, maxBuffer int) *batcher {
	return newBatcher(flush, flushSize, flushInterval, maxBuffer)
}

// Batcher is the exported type alias for batcher.
type Batcher = batcher
