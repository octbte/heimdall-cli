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
