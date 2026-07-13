package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Backup struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Size      int64     `json:"size"`
}

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

func (s *Store) ListBackups() ([]Backup, error) {
	backups := make([]Backup, 0)
	err := filepath.WalkDir(s.backupDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			return nil
		}
		relative, err := filepath.Rel(s.backupDir, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		backups = append(backups, Backup{ID: filepath.ToSlash(relative), CreatedAt: info.ModTime(), Size: info.Size()})
		return nil
	})
	if os.IsNotExist(err) {
		return []Backup{}, nil
	}
	if err != nil {
		return nil, err
	}
	sort.Slice(backups, func(i, j int) bool { return backups[i].CreatedAt.After(backups[j].CreatedAt) })
	return backups, nil
}
func (s *Store) Restore(id string) (Config, error) {
	data, cfg, err := s.LoadBackup(id)
	if err != nil {
		return Config{}, err
	}
	_, err = s.Save(data)
	return cfg, err
}

func (s *Store) LoadBackup(id string) ([]byte, Config, error) {
	clean := filepath.Clean(filepath.FromSlash(id))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return nil, Config{}, fmt.Errorf("invalid backup id")
	}
	path := filepath.Join(s.backupDir, clean)
	relative, err := filepath.Rel(s.backupDir, path)
	if err != nil || strings.HasPrefix(relative, "..") {
		return nil, Config{}, fmt.Errorf("invalid backup id")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, Config{}, fmt.Errorf("read backup: %w", err)
	}
	cfg, err := Load(bytes.NewReader(data))
	if err != nil {
		return nil, Config{}, err
	}
	return data, cfg, nil
}
