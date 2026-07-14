package cluster

import (
	"context"
	"errors"
	"testing"

	"github.com/nightzjp/kafka-manager/internal/config"
)

type fakeClient struct{ closed bool }

func (f *fakeClient) Ping(context.Context) error { return nil }
func (f *fakeClient) Close()                     { f.closed = true }

type fakeFactory struct {
	next Client
	err  error
}

type routingFactory struct {
	oldStarted chan struct{}
	releaseOld chan struct{}
	old        Client
	new        Client
}

type perClusterFactory struct {
	creates map[string]int
	clients map[string][]*fakeClient
}

type barrierFactory struct {
	started chan struct{}
	release chan struct{}
}

func (f barrierFactory) Create(context.Context, config.ClusterConfig) (Client, error) {
	f.started <- struct{}{}
	<-f.release
	return &fakeClient{}, nil
}

func (f *perClusterFactory) Create(_ context.Context, cfg config.ClusterConfig) (Client, error) {
	f.creates[cfg.ID]++
	client := &fakeClient{}
	f.clients[cfg.ID] = append(f.clients[cfg.ID], client)
	return client, nil
}

func (f routingFactory) Create(_ context.Context, cfg config.ClusterConfig) (Client, error) {
	if len(cfg.Brokers) > 0 && cfg.Brokers[0] == "old:9092" {
		close(f.oldStarted)
		<-f.releaseOld
		return f.old, nil
	}
	return f.new, nil
}

func (f fakeFactory) Create(context.Context, config.ClusterConfig) (Client, error) {
	return f.next, f.err
}

func TestManagerReplacesClientAfterSuccessfulConnection(t *testing.T) {
	oldClient, newClient := &fakeClient{}, &fakeClient{}
	m := NewManager(fakeFactory{next: oldClient})
	cfg := config.ClusterConfig{ID: "dev", Name: "Dev", Brokers: []string{"old:9092"}}
	if err := m.Apply(context.Background(), []config.ClusterConfig{cfg}); err != nil {
		t.Fatal(err)
	}
	m.factory = fakeFactory{next: newClient}
	updated := cfg
	updated.Brokers = []string{"new:9092"}
	if err := m.Apply(context.Background(), []config.ClusterConfig{updated}); err != nil {
		t.Fatal(err)
	}
	if !oldClient.closed {
		t.Fatal("old client was not closed")
	}
	if got, ok := m.Get("dev"); !ok || got != newClient {
		t.Fatal("new client not installed")
	}
}

func TestManagerKeepsOldClientWhenReplacementFails(t *testing.T) {
	oldClient := &fakeClient{}
	m := NewManager(fakeFactory{next: oldClient})
	cfg := config.ClusterConfig{ID: "dev", Name: "Dev", Brokers: []string{"old:9092"}}
	if err := m.Apply(context.Background(), []config.ClusterConfig{cfg}); err != nil {
		t.Fatal(err)
	}
	m.factory = fakeFactory{err: errors.New("connection refused")}
	updated := cfg
	updated.Brokers = []string{"new:9092"}
	if err := m.Apply(context.Background(), []config.ClusterConfig{updated}); err == nil {
		t.Fatal("expected replacement error")
	}
	if got, ok := m.Get("dev"); !ok || got != oldClient || oldClient.closed {
		t.Fatal("old client was not preserved")
	}
}

func TestManagerRemovesMissingCluster(t *testing.T) {
	client := &fakeClient{}
	m := NewManager(fakeFactory{next: client})
	if err := m.Apply(context.Background(), []config.ClusterConfig{{ID: "dev"}}); err != nil {
		t.Fatal(err)
	}
	if err := m.Apply(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Get("dev"); ok || !client.closed {
		t.Fatal("removed cluster remains active")
	}
}

func TestManagerUpsertKeepsOtherClusters(t *testing.T) {
	dev, test := &fakeClient{}, &fakeClient{}
	m := NewManager(fakeFactory{next: dev})
	if err := m.Upsert(context.Background(), config.ClusterConfig{ID: "dev"}); err != nil {
		t.Fatal(err)
	}
	m.factory = fakeFactory{next: test}
	if err := m.Upsert(context.Background(), config.ClusterConfig{ID: "test"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Get("dev"); !ok {
		t.Fatal("upsert removed existing cluster")
	}
	if got, ok := m.Get("test"); !ok || got != test {
		t.Fatal("new cluster not installed")
	}
}

func TestManagerRetainClosesClientsOutsideConfiguredSet(t *testing.T) {
	dev, removed := &fakeClient{}, &fakeClient{}
	m := NewManager(fakeFactory{next: dev})
	if err := m.Upsert(context.Background(), config.ClusterConfig{ID: "dev"}); err != nil {
		t.Fatal(err)
	}
	m.factory = fakeFactory{next: removed}
	if err := m.Upsert(context.Background(), config.ClusterConfig{ID: "removed"}); err != nil {
		t.Fatal(err)
	}

	m.Retain(map[string]struct{}{"dev": {}})

	if _, ok := m.Get("removed"); ok || !removed.closed {
		t.Fatal("client removed from configuration remains active")
	}
	if got, ok := m.Get("dev"); !ok || got != dev || dev.closed {
		t.Fatal("configured client was not retained")
	}
}

func TestManagerDiscardsUpsertStartedBeforeSuccessfulApply(t *testing.T) {
	oldClient, newClient := &fakeClient{}, &fakeClient{}
	factory := routingFactory{oldStarted: make(chan struct{}), releaseOld: make(chan struct{}), old: oldClient, new: newClient}
	m := NewManager(factory)
	upsertDone := make(chan error, 1)
	go func() {
		upsertDone <- m.Upsert(context.Background(), config.ClusterConfig{ID: "dev", Brokers: []string{"old:9092"}})
	}()
	<-factory.oldStarted

	if err := m.Apply(context.Background(), []config.ClusterConfig{{ID: "dev", Brokers: []string{"new:9092"}}}); err != nil {
		t.Fatal(err)
	}
	close(factory.releaseOld)
	if err := <-upsertDone; !errors.Is(err, ErrStaleConfiguration) {
		t.Fatalf("stale Upsert error = %v, want ErrStaleConfiguration", err)
	}
	if got, ok := m.Get("dev"); !ok || got != newClient {
		t.Fatal("stale Upsert replaced the newly applied client")
	}
	if !oldClient.closed {
		t.Fatal("discarded stale client was not closed")
	}
}

func TestManagerRejectsUpsertStartedAfterClusterRemoval(t *testing.T) {
	oldClient, ghost := &fakeClient{}, &fakeClient{}
	m := NewManager(fakeFactory{next: oldClient})
	if err := m.Apply(context.Background(), []config.ClusterConfig{{ID: "dev", Brokers: []string{"old:9092"}}}); err != nil {
		t.Fatal(err)
	}
	if err := m.Apply(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	m.factory = fakeFactory{next: ghost}

	err := m.Upsert(context.Background(), config.ClusterConfig{ID: "dev", Brokers: []string{"old:9092"}})
	if !errors.Is(err, ErrStaleConfiguration) {
		t.Fatalf("removed cluster Upsert error = %v, want ErrStaleConfiguration", err)
	}
	if _, ok := m.Get("dev"); ok {
		t.Fatal("removed cluster was recreated")
	}
}

func TestManagerMatchesConnectionConfigurationOnly(t *testing.T) {
	client := &fakeClient{}
	m := NewManager(fakeFactory{next: client})
	base := config.ClusterConfig{ID: "dev", Name: "old name", Brokers: []string{"broker:9092"}, ReadOnly: false}
	if err := m.Apply(context.Background(), []config.ClusterConfig{base}); err != nil {
		t.Fatal(err)
	}
	metadataOnly := base
	metadataOnly.Name = "new name"
	metadataOnly.ReadOnly = true
	if !m.Matches([]config.ClusterConfig{metadataOnly}) {
		t.Fatal("metadata-only change should not require reconnect")
	}
	changed := metadataOnly
	changed.Brokers = []string{"new:9092"}
	if m.Matches([]config.ClusterConfig{changed}) {
		t.Fatal("connection change was reported as already active")
	}
}

func TestManagerApplyReconnectsOnlyChangedCluster(t *testing.T) {
	factory := &perClusterFactory{creates: map[string]int{}, clients: map[string][]*fakeClient{}}
	m := NewManager(factory)
	initial := []config.ClusterConfig{
		{ID: "dev", Brokers: []string{"dev-old:9092"}},
		{ID: "test", Brokers: []string{"test:9092"}},
	}
	if err := m.Apply(context.Background(), initial); err != nil {
		t.Fatal(err)
	}
	unchangedTest := factory.clients["test"][0]
	updated := append([]config.ClusterConfig(nil), initial...)
	updated[0].Brokers = []string{"dev-new:9092"}
	if err := m.Apply(context.Background(), updated); err != nil {
		t.Fatal(err)
	}

	if factory.creates["dev"] != 2 || factory.creates["test"] != 1 {
		t.Fatalf("create counts = %+v, want dev=2 test=1", factory.creates)
	}
	if got, ok := m.Get("test"); !ok || got != unchangedTest || unchangedTest.closed {
		t.Fatal("unchanged cluster client was replaced or closed")
	}
}

func TestManagerAllowsConcurrentUpsertsForDesiredClusters(t *testing.T) {
	factory := barrierFactory{started: make(chan struct{}, 2), release: make(chan struct{})}
	m := NewManager(factory)
	m.SetDesired([]config.ClusterConfig{{ID: "dev"}, {ID: "test"}})
	errors := make(chan error, 2)
	go func() { errors <- m.Upsert(context.Background(), config.ClusterConfig{ID: "dev"}) }()
	go func() { errors <- m.Upsert(context.Background(), config.ClusterConfig{ID: "test"}) }()
	<-factory.started
	<-factory.started
	close(factory.release)
	for range 2 {
		if err := <-errors; err != nil {
			t.Fatalf("concurrent desired Upsert failed: %v", err)
		}
	}
	if _, ok := m.Get("dev"); !ok {
		t.Fatal("dev client missing")
	}
	if _, ok := m.Get("test"); !ok {
		t.Fatal("test client missing")
	}
}
