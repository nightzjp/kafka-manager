package cluster

import (
	"context"
	"fmt"
	"sync"

	"github.com/nightzjp/kafka-manager/internal/config"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Client interface {
	Ping(context.Context) error
	Close()
}

type Factory interface {
	Create(context.Context, config.ClusterConfig) (Client, error)
}

type Manager struct {
	mu      sync.RWMutex
	clients map[string]Client
	factory Factory
}

func NewManager(factory Factory) *Manager {
	return &Manager{clients: make(map[string]Client), factory: factory}
}

func (m *Manager) Apply(ctx context.Context, clusters []config.ClusterConfig) error {
	staged := make(map[string]Client, len(clusters))
	for _, cfg := range clusters {
		if cfg.Enabled != nil && !*cfg.Enabled {
			continue
		}
		client, err := m.factory.Create(ctx, cfg)
		if err != nil {
			closeClients(staged)
			return fmt.Errorf("connect cluster %s: %w", cfg.ID, err)
		}
		if err := client.Ping(ctx); err != nil {
			client.Close()
			closeClients(staged)
			return fmt.Errorf("ping cluster %s: %w", cfg.ID, err)
		}
		staged[cfg.ID] = client
	}
	m.mu.Lock()
	old := m.clients
	m.clients = staged
	m.mu.Unlock()
	closeClients(old)
	return nil
}

func (m *Manager) Upsert(ctx context.Context, cfg config.ClusterConfig) error {
	client, err := m.factory.Create(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect cluster %s: %w", cfg.ID, err)
	}
	if err := client.Ping(ctx); err != nil {
		client.Close()
		return fmt.Errorf("ping cluster %s: %w", cfg.ID, err)
	}
	m.mu.Lock()
	old := m.clients[cfg.ID]
	m.clients[cfg.ID] = client
	m.mu.Unlock()
	if old != nil {
		old.Close()
	}
	return nil
}

func (m *Manager) Get(id string) (Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.clients[id]
	return c, ok
}

func (m *Manager) Kafka(id string) (*kgo.Client, bool) {
	client, ok := m.Get(id)
	if !ok {
		return nil, false
	}
	kafkaClient, ok := client.(*KafkaClient)
	if !ok {
		return nil, false
	}
	return kafkaClient.Client, true
}

func (m *Manager) Close() {
	m.mu.Lock()
	old := m.clients
	m.clients = make(map[string]Client)
	m.mu.Unlock()
	closeClients(old)
}

func closeClients(clients map[string]Client) {
	for _, client := range clients {
		client.Close()
	}
}
