package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Store struct {
	path, backupDir string
	mu              sync.Mutex
	now             func() time.Time
}

func NewStore(path, backupDir string) *Store {
	return &Store{path: path, backupDir: backupDir, now: time.Now}
}

func (s *Store) Load() (Config, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}
	return Load(bytes.NewReader(data))
}

func (s *Store) Save(data []byte) (Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, err := Load(bytes.NewReader(data))
	if err != nil {
		return Config{}, err
	}
	old, err := os.ReadFile(s.path)
	if err != nil && !os.IsNotExist(err) {
		return Config{}, err
	}
	if len(old) > 0 {
		dir := filepath.Join(s.backupDir, s.now().Format("2006-01-02"))
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return Config{}, err
		}
		name := filepath.Join(dir, s.now().Format("150405.000000000")+".yaml")
		if err := os.WriteFile(name, old, 0o600); err != nil {
			return Config{}, fmt.Errorf("backup config: %w", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return Config{}, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".config-*.tmp")
	if err != nil {
		return Config{}, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return Config{}, err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return Config{}, err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return Config{}, err
	}
	if err := tmp.Close(); err != nil {
		return Config{}, err
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return Config{}, fmt.Errorf("replace config: %w", err)
	}
	return cfg, nil
}
