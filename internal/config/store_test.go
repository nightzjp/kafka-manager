package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEncryptDecrypt(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	ciphertext, err := Encrypt(key, "kafka-secret")
	if err != nil { t.Fatal(err) }
	if !strings.HasPrefix(ciphertext, "enc:v1:") || strings.Contains(ciphertext, "kafka-secret") { t.Fatalf("unsafe ciphertext %q", ciphertext) }
	plaintext, err := Decrypt(key, ciphertext)
	if err != nil || plaintext != "kafka-secret" { t.Fatalf("Decrypt() = %q, %v", plaintext, err) }
	if _, err := Decrypt([]byte("abcdef0123456789abcdef0123456789"), ciphertext); err == nil { t.Fatal("wrong key accepted") }
}

func TestStoreSaveCreatesBackupAndReloads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	initial := []byte("server:\n  username: admin\n  passwordHash: hash\nclusters:\n  - id: dev\n    name: Dev\n    brokers: [\"localhost:9092\"]\n")
	if err := os.WriteFile(path, initial, 0o600); err != nil { t.Fatal(err) }
	store := NewStore(path, filepath.Join(dir, "backups"))
	store.now = func() time.Time { return time.Date(2026, 7, 13, 10, 30, 0, 0, time.Local) }

	updated := strings.Replace(string(initial), "localhost:9092", "localhost:9093", 1)
	if _, err := store.Save([]byte(updated)); err != nil { t.Fatalf("Save() error = %v", err) }
	loaded, err := store.Load()
	if err != nil || loaded.Clusters[0].Brokers[0] != "localhost:9093" { t.Fatalf("Load() = %+v, %v", loaded, err) }
	backups, err := filepath.Glob(filepath.Join(dir, "backups", "2026-07-13", "*.yaml"))
	if err != nil || len(backups) != 1 { t.Fatalf("backups = %v, %v", backups, err) }
}

func TestStoreRejectsInvalidUpdate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	original := []byte("server:\n  username: admin\n  passwordHash: hash\nclusters:\n  - id: dev\n    name: Dev\n    brokers: [\"localhost:9092\"]\n")
	if err := os.WriteFile(path, original, 0o600); err != nil { t.Fatal(err) }
	store := NewStore(path, filepath.Join(dir, "backups"))
	if _, err := store.Save([]byte("invalid: true\n")); err == nil { t.Fatal("invalid config accepted") }
	after, _ := os.ReadFile(path)
	if string(after) != string(original) { t.Fatal("invalid update replaced valid config") }
}

func TestRuntimeDecryptsKafkaPasswords(t *testing.T){key:=[]byte("0123456789abcdef0123456789abcdef");encrypted,err:=Encrypt(key,"secret");if err!=nil{t.Fatal(err)};cfg:=Config{Clusters:[]ClusterConfig{{ID:"test",Security:SecurityConfig{Password:encrypted}}}};runtime,err:=Runtime(cfg,key);if err!=nil{t.Fatal(err)};if runtime.Clusters[0].Security.Password!="secret"{t.Fatalf("password=%q",runtime.Clusters[0].Security.Password)};if cfg.Clusters[0].Security.Password!=encrypted{t.Fatal("Runtime mutated persisted config")}}
