package dashboard

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nightzjp/kafka-manager/internal/config"
)

func TestSamplerStartsWithLoadingSnapshots(t *testing.T) {
	sampler := NewSampler([]config.ClusterConfig{{ID: "dev", Name: "开发环境"}}, Options{}, SourceFunc(func(context.Context, config.ClusterConfig) Snapshot {
		return Snapshot{}
	}))

	items, _ := sampler.Read()
	if len(items) != 1 || items[0].ID != "dev" || items[0].Status != StatusLoading || items[0].Online {
		t.Fatalf("items = %+v", items)
	}
}

func TestSamplerRefreshesClustersConcurrently(t *testing.T) {
	started := make(chan string, 2)
	release := make(chan struct{})
	source := SourceFunc(func(ctx context.Context, cfg config.ClusterConfig) Snapshot {
		started <- cfg.ID
		select {
		case <-release:
			return Snapshot{Status: StatusOnline, Online: true}
		case <-ctx.Done():
			return Snapshot{Status: StatusOffline, Error: ctx.Err().Error()}
		}
	})
	sampler := NewSampler([]config.ClusterConfig{{ID: "dev"}, {ID: "test"}}, Options{MaxConcurrent: 2}, source)
	done := make(chan struct{})
	go func() { sampler.Refresh(context.Background()); close(done) }()

	seen := map[string]bool{}
	for len(seen) < 2 {
		select {
		case id := <-started:
			seen[id] = true
		case <-time.After(time.Second):
			t.Fatalf("clusters were not sampled concurrently: %+v", seen)
		}
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("refresh did not finish")
	}
}

func TestSamplerRefreshNeverOverlaps(t *testing.T) {
	var active atomic.Int32
	var maximum atomic.Int32
	release := make(chan struct{})
	source := SourceFunc(func(context.Context, config.ClusterConfig) Snapshot {
		current := active.Add(1)
		for {
			old := maximum.Load()
			if current <= old || maximum.CompareAndSwap(old, current) {
				break
			}
		}
		<-release
		active.Add(-1)
		return Snapshot{Status: StatusOnline, Online: true}
	})
	sampler := NewSampler([]config.ClusterConfig{{ID: "dev"}}, Options{}, source)

	var calls sync.WaitGroup
	calls.Add(2)
	go func() { defer calls.Done(); sampler.Refresh(context.Background()) }()
	go func() { defer calls.Done(); sampler.Refresh(context.Background()) }()
	time.Sleep(20 * time.Millisecond)
	close(release)
	calls.Wait()
	if got := maximum.Load(); got != 1 {
		t.Fatalf("maximum overlapping refreshes = %d, want 1", got)
	}
}

func TestSamplerRunsWithoutBrowserRequestsAndCapsHistory(t *testing.T) {
	var samples atomic.Int32
	source := SourceFunc(func(context.Context, config.ClusterConfig) Snapshot {
		n := samples.Add(1)
		return Snapshot{Status: StatusOnline, Online: true, Topics: int(n), Partitions: int(n), LagAvailable: true}
	})
	sampler := NewSampler([]config.ClusterConfig{{ID: "dev"}}, Options{Interval: 5 * time.Millisecond, HistoryPoints: 2}, source)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { sampler.Run(ctx); close(done) }()

	deadline := time.Now().Add(time.Second)
	for samples.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	<-done
	if samples.Load() < 3 {
		t.Fatalf("background samples = %d, want at least 3", samples.Load())
	}
	_, history := sampler.Read()
	if got := len(history["dev"]); got != 2 {
		t.Fatalf("history points = %d, want 2", got)
	}
}

func TestSamplerDoesNotDuplicateInitialRefresh(t *testing.T) {
	var samples atomic.Int32
	sampler := NewSampler([]config.ClusterConfig{{ID: "dev"}}, Options{Interval: time.Hour}, SourceFunc(func(context.Context, config.ClusterConfig) Snapshot {
		samples.Add(1)
		return Snapshot{Status: StatusOnline, Online: true}
	}))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { sampler.Run(ctx); close(done) }()
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done
	if got := samples.Load(); got != 1 {
		t.Fatalf("initial samples = %d, want 1", got)
	}
}

func TestSamplerUpdateReusesExistingSnapshotAndAddsLoadingCluster(t *testing.T) {
	sampler := NewSampler([]config.ClusterConfig{{ID: "dev", Name: "开发环境"}}, Options{}, SourceFunc(func(context.Context, config.ClusterConfig) Snapshot {
		return Snapshot{Status: StatusOnline, Online: true, Topics: 7}
	}))
	sampler.Refresh(context.Background())
	sampler.Update([]config.ClusterConfig{{ID: "dev", Name: "开发集群"}, {ID: "test", Name: "测试环境"}}, Options{})

	items, _ := sampler.Read()
	if len(items) != 2 {
		t.Fatalf("items = %+v", items)
	}
	if items[0].Name != "开发集群" || items[0].Topics != 7 || items[0].Status != StatusOnline {
		t.Fatalf("existing snapshot = %+v", items[0])
	}
	if items[1].ID != "test" || items[1].Status != StatusLoading {
		t.Fatalf("new snapshot = %+v", items[1])
	}
}

func TestSamplerConnectionChangeClearsOfflineBackoff(t *testing.T) {
	var calls atomic.Int32
	source := SourceFunc(func(context.Context, config.ClusterConfig) Snapshot {
		calls.Add(1)
		return Snapshot{Status: StatusOffline, Error: "broker unavailable"}
	})
	sampler := NewSampler([]config.ClusterConfig{{ID: "dev", Brokers: []string{"old:9092"}}}, Options{Interval: time.Hour}, source)
	sampler.Refresh(context.Background())
	sampler.Update([]config.ClusterConfig{{ID: "dev", Brokers: []string{"new:9092"}}}, Options{Interval: time.Hour})
	sampler.Refresh(context.Background())

	if got := calls.Load(); got != 2 {
		t.Fatalf("source calls after connection change = %d, want 2", got)
	}
}

func TestSamplerDiscardsInFlightResultAfterConnectionChange(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	sampler := NewSampler([]config.ClusterConfig{{ID: "dev", Brokers: []string{"old:9092"}}}, Options{}, SourceFunc(func(context.Context, config.ClusterConfig) Snapshot {
		close(started)
		<-release
		return Snapshot{Status: StatusOnline, Online: true, Topics: 99}
	}))
	done := make(chan struct{})
	go func() { sampler.Refresh(context.Background()); close(done) }()
	<-started

	sampler.Update([]config.ClusterConfig{{ID: "dev", Brokers: []string{"new:9092"}}}, Options{})
	close(release)
	<-done

	items, _ := sampler.Read()
	if len(items) != 1 || items[0].Status != StatusLoading || items[0].Topics != 0 {
		t.Fatalf("stale in-flight result replaced new connection state: %+v", items)
	}
}

func TestSamplerAppliesCurrentMetadataToInFlightResult(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	sampler := NewSampler([]config.ClusterConfig{{ID: "dev", Name: "旧名称", Brokers: []string{"broker:9092"}}}, Options{}, SourceFunc(func(context.Context, config.ClusterConfig) Snapshot {
		close(started)
		<-release
		return Snapshot{Status: StatusOnline, Online: true, Topics: 9}
	}))
	done := make(chan struct{})
	go func() { sampler.Refresh(context.Background()); close(done) }()
	<-started

	sampler.Update([]config.ClusterConfig{{ID: "dev", Name: "新名称", Brokers: []string{"broker:9092"}, ReadOnly: true}}, Options{})
	close(release)
	<-done

	items, _ := sampler.Read()
	if len(items) != 1 || items[0].Name != "新名称" || !items[0].ReadOnly || items[0].Topics != 9 {
		t.Fatalf("in-flight result did not retain current metadata: %+v", items)
	}
}

func TestSamplerBacksOffOfflineCluster(t *testing.T) {
	var calls atomic.Int32
	sampler := NewSampler([]config.ClusterConfig{{ID: "dev"}}, Options{Interval: time.Second}, SourceFunc(func(context.Context, config.ClusterConfig) Snapshot {
		calls.Add(1)
		return Snapshot{Status: StatusOffline, Error: "broker unavailable"}
	}))

	sampler.Refresh(context.Background())
	sampler.Refresh(context.Background())
	if got := calls.Load(); got != 1 {
		t.Fatalf("offline source calls = %d, want 1 while retry is backed off", got)
	}
}

func TestSamplerDoesNotRecordUnavailableLagAsZero(t *testing.T) {
	sampler := NewSampler([]config.ClusterConfig{{ID: "dev"}}, Options{}, SourceFunc(func(context.Context, config.ClusterConfig) Snapshot {
		return Snapshot{Status: StatusOnline, Online: true, LagAvailable: false}
	}))

	sampler.Refresh(context.Background())
	_, history := sampler.Read()
	if got := len(history["dev"]); got != 0 {
		t.Fatalf("history contains %d false zero-lag points, want 0", got)
	}
}

func TestFailureBackoffIsExponentialAndCapped(t *testing.T) {
	base := 15 * time.Second
	if got := failureBackoff(base, 1); got != 30*time.Second {
		t.Fatalf("first backoff = %v", got)
	}
	if got := failureBackoff(base, 2); got != time.Minute {
		t.Fatalf("second backoff = %v", got)
	}
	if got := failureBackoff(base, 20); got != 5*time.Minute {
		t.Fatalf("capped backoff = %v", got)
	}
}

func BenchmarkSamplerRead(b *testing.B) {
	configs := make([]config.ClusterConfig, 20)
	for index := range configs {
		configs[index] = config.ClusterConfig{ID: string(rune('a' + index)), Name: "cluster"}
	}
	sampler := NewSampler(configs, Options{HistoryPoints: 240}, SourceFunc(func(context.Context, config.ClusterConfig) Snapshot {
		return Snapshot{Status: StatusOnline, Online: true, Topics: 100, Partitions: 1000, LagAvailable: true}
	}))
	for index := 0; index < 240; index++ {
		sampler.Refresh(context.Background())
	}
	b.ResetTimer()
	b.RunParallel(func(parallel *testing.PB) {
		for parallel.Next() {
			sampler.Read()
		}
	})
}
