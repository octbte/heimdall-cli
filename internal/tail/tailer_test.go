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
	timeout := time.After(3 * time.Second)
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
	if err := os.WriteFile(path, []byte("old content\n"), 0600); err != nil {
		t.Fatal(err)
	}

	store := &offsetStore{dir: dir}
	lines := make(chan Line, 20)

	tl := newTailer(path, "app.log", store, lines)
	go tl.run()
	defer tl.stop()

	// drain initial content (tailer seeks to end on first run, so no lines expected from "old content")
	time.Sleep(150 * time.Millisecond)
	for len(lines) > 0 {
		<-lines
	}

	// simulate rotation: remove old file and create new one with same name
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("new file line\n"), 0600); err != nil {
		t.Fatal(err)
	}

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

func TestTailerDetectsRotationWithDelay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	if err := os.WriteFile(path, []byte("old content\n"), 0600); err != nil {
		t.Fatal(err)
	}

	store := &offsetStore{dir: dir}
	lines := make(chan Line, 20)

	tl := newTailer(path, "app.log", store, lines)
	go tl.run()
	defer tl.stop()

	time.Sleep(150 * time.Millisecond)
	for len(lines) > 0 {
		<-lines
	}

	// simulate compression gap: old file removed, new file appears after 800ms
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(800 * time.Millisecond)
		_ = os.WriteFile(path, []byte("after compression gap\n"), 0600)
	}()

	var got Line
	timeout := time.After(5 * time.Second)
	select {
	case got = <-lines:
	case <-timeout:
		t.Fatal("timeout: tailer did not recover after rotation gap")
	}
	if got.Message != "after compression gap" {
		t.Errorf("message=%q, want 'after compression gap'", got.Message)
	}
}

func TestTailerRotationDuringDowntime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")

	// Write old content at a large offset that will be saved as the offset.
	oldContent := "old line\n"
	if err := os.WriteFile(path, []byte(oldContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Pre-seed the offset store with a large offset (simulating the tailer
	// having read a large file before downtime).
	store := &offsetStore{dir: dir}
	key := store.key(path)
	if err := store.save(key, 9999); err != nil {
		t.Fatal(err)
	}

	// Simulate rotation during downtime: replace file with smaller new content.
	if err := os.WriteFile(path, []byte("new line after downtime\n"), 0600); err != nil {
		t.Fatal(err)
	}

	lines := make(chan Line, 10)
	tl := newTailer(path, "app.log", store, lines)
	go tl.run()
	defer tl.stop()

	var got Line
	timeout := time.After(3 * time.Second)
	select {
	case got = <-lines:
	case <-timeout:
		t.Fatal("timeout: tailer did not recover after rotation during downtime")
	}
	if got.Message != "new line after downtime" {
		t.Errorf("message=%q, want 'new line after downtime'", got.Message)
	}
}
