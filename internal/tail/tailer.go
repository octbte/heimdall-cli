package tail

import (
	"bufio"
	"io"
	"log"
	"os"
	"sync"
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
	stopOnce   sync.Once
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
	t.stopOnce.Do(func() { close(t.stopCh) })
}

func (t *tailer) run() {
	key := t.store.key(t.path)
	offset, err := t.store.load(key)
	if err != nil {
		log.Printf("tail: could not load offset for %s: %v", t.path, err)
	}

	// seekEnd=true only on first run (no saved offset file); offset==0 after rotation means start of new file.
	f, inode, err := t.openAt(offset, !t.store.exists(key))
	if err != nil {
		log.Printf("tail: could not open %s: %v", t.path, err)
		return
	}
	defer f.Close()

	// Detect rotation during downtime: if saved offset exceeds current file size,
	// the file was replaced while the tailer was stopped. Reset to byte 0.
	if offset > 0 {
		if info, statErr := f.Stat(); statErr == nil && info.Size() < offset {
			log.Printf("tail: saved offset %d exceeds file size %d for %s; rotation during downtime, resetting", offset, info.Size(), t.path)
			if _, seekErr := f.Seek(0, io.SeekStart); seekErr == nil {
				offset = 0
				t.store.delete(key)
			}
		}
	}

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
						select {
						case t.out <- Line{
							SourceName: t.sourceName,
							Message:    msg,
							Level:      detectLevel(msg),
							OccurredAt: time.Now().UTC(),
						}:
						case <-t.stopCh:
							return
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
				// Retry reopening: the new file may not exist yet during the brief
				// window between the old file being renamed and the new one created.
				var newF *os.File
				var newIno uint64
				var reopenErr error
				for attempt := 0; attempt < 10; attempt++ {
					newF, newIno, reopenErr = t.openAt(0, false)
					if reopenErr == nil {
						break
					}
					select {
					case <-t.stopCh:
						return
					case <-time.After(500 * time.Millisecond):
					}
				}
				if reopenErr != nil {
					log.Printf("tail: could not reopen %s after rotation: %v", t.path, reopenErr)
					return
				}
				f, inode = newF, newIno
				reader = bufio.NewReader(f)
				offset = 0
				t.store.delete(key) // delete so next restart reads from byte 0, not skips to end
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
