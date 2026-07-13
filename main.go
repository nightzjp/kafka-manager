package main

import (
	"context"
	"crypto/rand"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nightzjp/kafka-manager/internal/api"
	"github.com/nightzjp/kafka-manager/internal/audit"
	"github.com/nightzjp/kafka-manager/internal/auth"
	"github.com/nightzjp/kafka-manager/internal/cluster"
	"github.com/nightzjp/kafka-manager/internal/config"
	"github.com/nightzjp/kafka-manager/internal/webassets"
)

func main() {
	configPath := flag.String("config", "./config.yaml", "path to YAML configuration")
	printHash := flag.Bool("print-password-hash", false, "print Argon2id hash of KAFKA_MANAGER_PASSWORD and exit")
	flag.Parse()
	if *printHash {
		password := os.Getenv("KAFKA_MANAGER_PASSWORD")
		if password == "" {
			log.Fatal("KAFKA_MANAGER_PASSWORD is required")
		}
		hash, err := auth.HashPassword(password)
		if err != nil {
			log.Fatal(err)
		}
		os.Stdout.WriteString(hash + "\n")
		return
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		log.Fatalf("initialize session key: %v", err)
	}
	store := config.NewStore(*configPath, filepath.Join(filepath.Dir(*configPath), "data", "config-backups"))
	persisted, err := store.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}
	cfg, err := config.Runtime(persisted, secret)
	if err != nil {
		log.Fatalf("decrypt configuration: %v", err)
	}
	cleanupBackups := func() {
		current, loadErr := store.Load()
		if loadErr != nil {
			log.Printf("config backup cleanup skipped: %v", loadErr)
			return
		}
		if cleanupErr := store.CleanupBackups(current.Audit.ConfigBackupRetentionDays, time.Now()); cleanupErr != nil {
			log.Printf("config backup cleanup: %v", cleanupErr)
		}
	}
	cleanupBackups()
	manager := cluster.NewManager(cluster.KafkaFactory{})
	defer manager.Close()
	connect := func(current config.Config) {
		for _, item := range current.Clusters {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := manager.Upsert(ctx, item)
			cancel()
			if err != nil {
				log.Printf("cluster %s offline: %v", item.ID, err)
			}
		}
	}
	connect(cfg)
	handler := api.NewServer(cfg, store, manager, secret)
	if cfg.Audit.Enabled == nil || *cfg.Audit.Enabled {
		auditWriter, err := audit.NewWriter(audit.Config{Directory: cfg.Audit.Directory, MaxFileSizeBytes: int64(cfg.Audit.MaxFileSizeMB) * 1024 * 1024, RetentionDays: cfg.Audit.RetentionDays})
		if err != nil {
			log.Fatalf("initialize audit log: %v", err)
		}
		defer auditWriter.Close()
		if err := audit.Cleanup(cfg.Audit.Directory, cfg.Audit.RetentionDays, time.Now()); err != nil {
			log.Printf("audit cleanup: %v", err)
		}
		handler.SetAudit(auditWriter, cfg.Audit.Directory)
	}
	server := &http.Server{
		Addr: cfg.Server.ListenAddress, Handler: webassets.Handler(handler),
		ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second,
	}
	watcher, err := config.NewWatcher(*configPath, func(updated config.Config) {
		runtimeCfg, decryptErr := config.Runtime(updated, secret)
		if decryptErr != nil {
			log.Printf("configuration reload rejected: %v", decryptErr)
			return
		}
		connect(runtimeCfg)
		handler.UpdateConfig(runtimeCfg)
		log.Printf("configuration reloaded")
	})
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := watcher.Poll(); err != nil {
					log.Printf("configuration reload rejected: %v", err)
				}
			}
		}
	}()
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanupBackups()
			}
		}
	}()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	log.Printf("kafka-manager listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
