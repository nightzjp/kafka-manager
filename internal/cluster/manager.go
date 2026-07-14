package cluster

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/nightzjp/kafka-manager/internal/config"
	"github.com/twmb/franz-go/pkg/kgo"
)

var ErrStaleConfiguration = errors.New("stale cluster configuration")

type Client interface {
	Ping(context.Context) error
	Close()
}

type Factory interface {
	Create(context.Context, config.ClusterConfig) (Client, error)
}

type Manager struct {
	mu         sync.RWMutex
	clients    map[string]Client
	configs    map[string]config.ClusterConfig
	desired    map[string]config.ClusterConfig
	desiredSet bool
	generation uint64
	factory    Factory
}

func NewManager(factory Factory) *Manager {
	return &Manager{clients: make(map[string]Client), configs: make(map[string]config.ClusterConfig), desired: make(map[string]config.ClusterConfig), factory: factory}
}

func (m *Manager) Apply(ctx context.Context, clusters []config.ClusterConfig) error {
	desired := enabledConfigs(clusters)
	m.mu.RLock()
	generation := m.generation
	reused := make(map[string]bool, len(desired))
	for id, cfg := range desired {
		active, configured := m.configs[id]
		_, connected := m.clients[id]
		reused[id] = configured && connected && sameConnection(active, cfg)
	}
	m.mu.RUnlock()

	created := make(map[string]Client, len(desired))
	for _, cfg := range clusters {
		if cfg.Enabled != nil && !*cfg.Enabled {
			continue
		}
		if reused[cfg.ID] {
			continue
		}
		client, err := m.factory.Create(ctx, cfg)
		if err != nil {
			closeClients(created)
			return fmt.Errorf("connect cluster %s: %w", cfg.ID, err)
		}
		if err := client.Ping(ctx); err != nil {
			client.Close()
			closeClients(created)
			return fmt.Errorf("ping cluster %s: %w", cfg.ID, err)
		}
		created[cfg.ID] = client
	}

	m.mu.Lock()
	if generation != m.generation {
		m.mu.Unlock()
		closeClients(created)
		return ErrStaleConfiguration
	}
	old := m.clients
	next := make(map[string]Client, len(desired))
	for id := range desired {
		if client, ok := created[id]; ok {
			next[id] = client
			continue
		}
		next[id] = old[id]
	}
	m.clients = next
	m.configs = desired
	m.desired = desired
	m.desiredSet = true
	m.generation++
	m.mu.Unlock()
	for id, client := range old {
		if !reused[id] {
			client.Close()
		}
	}
	return nil
}

func (m *Manager) Upsert(ctx context.Context, cfg config.ClusterConfig) error {
	m.mu.RLock()
	generation := m.generation
	allowed := m.desiredAllowsLocked(cfg)
	m.mu.RUnlock()
	if !allowed {
		return fmt.Errorf("connect cluster %s: %w", cfg.ID, ErrStaleConfiguration)
	}
	client, err := m.factory.Create(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect cluster %s: %w", cfg.ID, err)
	}
	if err := client.Ping(ctx); err != nil {
		client.Close()
		return fmt.Errorf("ping cluster %s: %w", cfg.ID, err)
	}
	m.mu.Lock()
	if generation != m.generation || !m.desiredAllowsLocked(cfg) {
		m.mu.Unlock()
		client.Close()
		return fmt.Errorf("connect cluster %s: %w", cfg.ID, ErrStaleConfiguration)
	}
	old := m.clients[cfg.ID]
	m.clients[cfg.ID] = client
	m.configs[cfg.ID] = cfg
	m.mu.Unlock()
	if old != nil {
		old.Close()
	}
	return nil
}

// SetDesired establishes the authoritative set of enabled cluster connection
// configurations. It closes clients that are no longer allowed before any
// background reconnect can recreate them.
func (m *Manager) SetDesired(clusters []config.ClusterConfig) {
	desired := enabledConfigs(clusters)
	removed := make(map[string]Client)
	m.mu.Lock()
	for id, client := range m.clients {
		active, activeOK := m.configs[id]
		wanted, wantedOK := desired[id]
		if wantedOK && activeOK && sameConnection(active, wanted) {
			continue
		}
		removed[id] = client
		delete(m.clients, id)
		delete(m.configs, id)
	}
	m.desired = desired
	m.desiredSet = true
	m.generation++
	m.mu.Unlock()
	closeClients(removed)
}

// Matches reports whether the requested Broker and security configuration is
// already authoritative. Display-only changes do not require a reconnect.
func (m *Manager) Matches(clusters []config.ClusterConfig) bool {
	enabled := enabledConfigs(clusters)
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.desiredSet || len(enabled) != len(m.desired) {
		return false
	}
	for id, cfg := range enabled {
		wanted, ok := m.desired[id]
		if !ok || !sameConnection(wanted, cfg) {
			return false
		}
	}
	return true
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

// Retain closes and removes clients whose IDs are not in the active
// configuration. Existing clients in the configured set are left untouched.
func (m *Manager) Retain(active map[string]struct{}) {
	removed := make(map[string]Client)
	m.mu.Lock()
	for id, client := range m.clients {
		if _, ok := active[id]; ok {
			continue
		}
		removed[id] = client
		delete(m.clients, id)
		delete(m.configs, id)
	}
	if len(removed) > 0 {
		m.generation++
	}
	m.mu.Unlock()
	closeClients(removed)
}

func (m *Manager) Close() {
	m.mu.Lock()
	old := m.clients
	m.clients = make(map[string]Client)
	m.configs = make(map[string]config.ClusterConfig)
	m.desired = make(map[string]config.ClusterConfig)
	m.desiredSet = true
	m.generation++
	m.mu.Unlock()
	closeClients(old)
}

func (m *Manager) desiredAllowsLocked(cfg config.ClusterConfig) bool {
	if !m.desiredSet {
		return true
	}
	wanted, ok := m.desired[cfg.ID]
	return ok && sameConnection(wanted, cfg)
}

func enabledConfigs(clusters []config.ClusterConfig) map[string]config.ClusterConfig {
	enabled := make(map[string]config.ClusterConfig, len(clusters))
	for _, cfg := range clusters {
		if cfg.Enabled != nil && !*cfg.Enabled {
			continue
		}
		enabled[cfg.ID] = cfg
	}
	return enabled
}

func sameConnection(left, right config.ClusterConfig) bool {
	return slices.Equal(left.Brokers, right.Brokers) && left.Security == right.Security
}

func closeClients(clients map[string]Client) {
	for _, client := range clients {
		client.Close()
	}
}
