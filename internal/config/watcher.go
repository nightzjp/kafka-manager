package config

import (
	"bytes"
	"crypto/sha256"
	"os"
)

type Watcher struct {
	path     string
	hash     [32]byte
	onChange func(Config)
}

func NewWatcher(path string, onChange func(Config)) (*Watcher, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if _, err := Load(bytes.NewReader(data)); err != nil {
		return nil, err
	}
	return &Watcher{path: path, hash: sha256.Sum256(data), onChange: onChange}, nil
}
func (w *Watcher) Poll() (bool, error) {
	data, err := os.ReadFile(w.path)
	if err != nil {
		return false, err
	}
	hash := sha256.Sum256(data)
	if hash == w.hash {
		return false, nil
	}
	cfg, err := Load(bytes.NewReader(data))
	if err != nil {
		return false, err
	}
	w.hash = hash
	if w.onChange != nil {
		w.onChange(cfg)
	}
	return true, nil
}
