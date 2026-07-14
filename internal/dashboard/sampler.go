package dashboard

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/nightzjp/kafka-manager/internal/config"
)

type Status string

const (
	StatusLoading Status = "loading"
	StatusOnline  Status = "online"
	StatusOffline Status = "offline"
)

type Snapshot struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          Status `json:"status"`
	Online          bool   `json:"online"`
	Error           string `json:"error,omitempty"`
	SampledAt       int64  `json:"sampledAt,omitempty"`
	LatencyMS       int64  `json:"latencyMs"`
	Brokers         int    `json:"brokers"`
	Topics          int    `json:"topics"`
	Partitions      int    `json:"partitions"`
	ConsumerGroups  int    `json:"consumerGroups"`
	UnderReplicated int    `json:"underReplicated"`
	TotalLag        int64  `json:"totalLag"`
	LagAvailable    bool   `json:"lagAvailable"`
	LagError        string `json:"lagError,omitempty"`
	ReadOnly        bool   `json:"readOnly"`
}

type Point struct {
	Timestamp  int64 `json:"timestamp"`
	TotalLag   int64 `json:"totalLag"`
	Partitions int   `json:"partitions"`
	Topics     int   `json:"topics"`
}

type Source interface {
	Snapshot(context.Context, config.ClusterConfig) Snapshot
}

type SourceFunc func(context.Context, config.ClusterConfig) Snapshot

func (f SourceFunc) Snapshot(ctx context.Context, cfg config.ClusterConfig) Snapshot {
	return f(ctx, cfg)
}

type Options struct {
	Interval      time.Duration
	HistoryPoints int
	MaxConcurrent int
}

type Sampler struct {
	mu        sync.RWMutex
	refreshMu sync.Mutex
	configs   []config.ClusterConfig
	options   Options
	source    Source
	items     map[string]Snapshot
	history   map[string][]Point
	retries   map[string]retryState
	wake      chan struct{}
}

type retryState struct {
	failures int
	after    time.Time
}

func NewSampler(configs []config.ClusterConfig, options Options, source Source) *Sampler {
	s := &Sampler{
		source:  source,
		items:   make(map[string]Snapshot),
		history: make(map[string][]Point),
		retries: make(map[string]retryState),
		wake:    make(chan struct{}, 1),
	}
	s.Update(configs, options)
	select {
	case <-s.wake:
	default:
	}
	return s
}

func (s *Sampler) Update(configs []config.ClusterConfig, options Options) {
	options = normalizeOptions(options)
	enabled := enabledClusters(configs)

	s.mu.Lock()
	previousConfigs := make(map[string]config.ClusterConfig, len(s.configs))
	for _, cfg := range s.configs {
		previousConfigs[cfg.ID] = cfg
	}
	nextItems := make(map[string]Snapshot, len(enabled))
	nextHistory := make(map[string][]Point, len(enabled))
	nextRetries := make(map[string]retryState, len(enabled))
	for _, cfg := range enabled {
		previous, existed := previousConfigs[cfg.ID]
		sameConnection := existed && sameConnection(previous, cfg)
		item, ok := s.items[cfg.ID]
		if !ok || !sameConnection {
			item = loadingSnapshot(cfg)
		}
		item.ID = cfg.ID
		item.Name = cfg.Name
		item.ReadOnly = cfg.ReadOnly
		nextItems[cfg.ID] = item
		var points []Point
		if sameConnection {
			points = append([]Point(nil), s.history[cfg.ID]...)
		}
		if len(points) > options.HistoryPoints {
			points = points[len(points)-options.HistoryPoints:]
		}
		nextHistory[cfg.ID] = points
		if retry, ok := s.retries[cfg.ID]; ok && sameConnection {
			nextRetries[cfg.ID] = retry
		}
	}
	s.configs = enabled
	s.options = options
	s.items = nextItems
	s.history = nextHistory
	s.retries = nextRetries
	s.mu.Unlock()

	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Sampler) Run(ctx context.Context) {
	for {
		s.Refresh(ctx)
		interval := s.interval()
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-s.wake:
			if !timer.Stop() {
				<-timer.C
			}
		case <-timer.C:
		}
	}
}

func (s *Sampler) Refresh(ctx context.Context) {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	if ctx.Err() != nil {
		return
	}

	s.mu.RLock()
	now := time.Now()
	configs := make([]config.ClusterConfig, 0, len(s.configs))
	for _, cfg := range s.configs {
		if retry, ok := s.retries[cfg.ID]; ok && retry.after.After(now) {
			continue
		}
		configs = append(configs, cfg)
	}
	maximum := s.options.MaxConcurrent
	s.mu.RUnlock()
	if len(configs) == 0 {
		return
	}

	type result struct {
		cfg      config.ClusterConfig
		snapshot Snapshot
	}
	results := make(chan result, len(configs))
	semaphore := make(chan struct{}, maximum)
	var workers sync.WaitGroup
	for _, cfg := range configs {
		cfg := cfg
		workers.Add(1)
		go func() {
			defer workers.Done()
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-semaphore }()
			snapshot := s.source.Snapshot(ctx, cfg)
			normalizeSnapshot(&snapshot, cfg)
			results <- result{cfg: cfg, snapshot: snapshot}
		}()
	}
	workers.Wait()
	close(results)

	s.mu.Lock()
	defer s.mu.Unlock()
	for sampled := range results {
		current, ok := s.currentConfigLocked(sampled.cfg)
		if !ok {
			continue
		}
		sampled.snapshot.Name = current.Name
		sampled.snapshot.ReadOnly = current.ReadOnly
		s.items[sampled.cfg.ID] = sampled.snapshot
		if sampled.snapshot.Status != StatusOnline {
			if sampled.snapshot.Status == StatusOffline {
				retry := s.retries[sampled.cfg.ID]
				retry.failures++
				retry.after = time.Now().Add(failureBackoff(s.options.Interval, retry.failures))
				s.retries[sampled.cfg.ID] = retry
			}
			continue
		}
		delete(s.retries, sampled.cfg.ID)
		if !sampled.snapshot.LagAvailable {
			continue
		}
		point := Point{Timestamp: sampled.snapshot.SampledAt, TotalLag: sampled.snapshot.TotalLag, Partitions: sampled.snapshot.Partitions, Topics: sampled.snapshot.Topics}
		points := append(s.history[sampled.cfg.ID], point)
		if len(points) > s.options.HistoryPoints {
			points = points[len(points)-s.options.HistoryPoints:]
		}
		s.history[sampled.cfg.ID] = points
	}
}

func (s *Sampler) Read() ([]Snapshot, map[string][]Point) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Snapshot, 0, len(s.configs))
	history := make(map[string][]Point, len(s.configs))
	for _, cfg := range s.configs {
		items = append(items, s.items[cfg.ID])
		history[cfg.ID] = append([]Point(nil), s.history[cfg.ID]...)
	}
	return items, history
}

func (s *Sampler) interval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.options.Interval
}

func (s *Sampler) currentConfigLocked(sampled config.ClusterConfig) (config.ClusterConfig, bool) {
	for _, cfg := range s.configs {
		if cfg.ID == sampled.ID {
			return cfg, sameConnection(cfg, sampled)
		}
	}
	return config.ClusterConfig{}, false
}

func sameConnection(left, right config.ClusterConfig) bool {
	return slices.Equal(left.Brokers, right.Brokers) && left.Security == right.Security
}

func normalizeOptions(options Options) Options {
	if options.Interval <= 0 {
		options.Interval = 15 * time.Second
	}
	if options.HistoryPoints < 2 {
		options.HistoryPoints = 240
	}
	if options.MaxConcurrent < 1 {
		options.MaxConcurrent = 4
	}
	return options
}

func failureBackoff(base time.Duration, failures int) time.Duration {
	if base <= 0 {
		base = 15 * time.Second
	}
	delay := base
	for i := 0; i < failures; i++ {
		if delay >= 5*time.Minute/2 {
			return 5 * time.Minute
		}
		delay *= 2
	}
	if delay > 5*time.Minute {
		return 5 * time.Minute
	}
	return delay
}

func enabledClusters(configs []config.ClusterConfig) []config.ClusterConfig {
	result := make([]config.ClusterConfig, 0, len(configs))
	for _, cfg := range configs {
		if cfg.Enabled != nil && !*cfg.Enabled {
			continue
		}
		result = append(result, cfg)
	}
	return result
}

func loadingSnapshot(cfg config.ClusterConfig) Snapshot {
	return Snapshot{ID: cfg.ID, Name: cfg.Name, Status: StatusLoading, ReadOnly: cfg.ReadOnly}
}

func normalizeSnapshot(snapshot *Snapshot, cfg config.ClusterConfig) {
	snapshot.ID = cfg.ID
	snapshot.Name = cfg.Name
	snapshot.ReadOnly = cfg.ReadOnly
	if snapshot.Status == "" {
		if snapshot.Online {
			snapshot.Status = StatusOnline
		} else {
			snapshot.Status = StatusOffline
		}
	}
	snapshot.Online = snapshot.Status == StatusOnline
	if snapshot.SampledAt == 0 {
		snapshot.SampledAt = time.Now().UnixMilli()
	}
}
