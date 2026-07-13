package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWatcherLoadsOnlyValidChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	base := "server:\n  username: admin\n  passwordHash: hash\nclusters:\n  - id: dev\n    name: Dev\n    brokers: [\"localhost:9092\"]\n"
	if err := os.WriteFile(path, []byte(base), 0o600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	watcher, err := NewWatcher(path, func(Config) { calls++ })
	if err != nil {
		t.Fatal(err)
	}
	if changed, err := watcher.Poll(); err != nil || changed {
		t.Fatalf("initial Poll = %v,%v", changed, err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(base, "9092", "9093", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if changed, err := watcher.Poll(); err != nil || !changed || calls != 1 {
		t.Fatalf("valid Poll = %v,%v calls=%d", changed, err, calls)
	}
	if err := os.WriteFile(path, []byte("invalid: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if changed, err := watcher.Poll(); err == nil || changed || calls != 1 {
		t.Fatalf("invalid Poll = %v,%v calls=%d", changed, err, calls)
	}
}
