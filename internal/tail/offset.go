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

func (s *offsetStore) delete(key string) {
	os.Remove(filepath.Join(s.dir, key+".json"))
}

// exists reports whether an offset file exists for the given key.
func (s *offsetStore) exists(key string) bool {
	_, err := os.Stat(filepath.Join(s.dir, key+".json"))
	return err == nil
}

// NewOffsetStore creates an offsetStore in ~/.heimdall/offsets.
func NewOffsetStore() (*offsetStore, error) {
	return newOffsetStore()
}
