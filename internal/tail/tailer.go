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

	// seekEnd=true on first run (offset==0 and no saved state), false after rotation
	f, inode, err := t.openAt(offset, offset == 0)
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
				f, inode, err = t.openAt(0, false)
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

// NewTailer creates a tailer. Use Run() in a goroutine.
func NewTailer(path, sourceName string, store *offsetStore, out chan<- Line) *tailer {
	return newTailer(path, sourceName, store, out)
}

// Tailer is the exported type alias for tailer.
type Tailer = tailer

func (t *tailer) Run()  { t.run() }
func (t *tailer) Stop() { t.stop() }
