package cluster

import (
	"context"
	"errors"
	"testing"

	"github.com/nightzjp/kafka-manager/internal/config"
)

type fakeClient struct { closed bool }
func (f *fakeClient) Ping(context.Context) error { return nil }
func (f *fakeClient) Close() { f.closed = true }

type fakeFactory struct { next Client; err error }
func (f fakeFactory) Create(context.Context, config.ClusterConfig) (Client, error) { return f.next, f.err }

func TestManagerReplacesClientAfterSuccessfulConnection(t *testing.T) {
	oldClient, newClient := &fakeClient{}, &fakeClient{}
	m := NewManager(fakeFactory{next: oldClient})
	cfg := config.ClusterConfig{ID: "dev", Name: "Dev", Brokers: []string{"localhost:9092"}}
	if err := m.Apply(context.Background(), []config.ClusterConfig{cfg}); err != nil { t.Fatal(err) }
	m.factory = fakeFactory{next: newClient}
	if err := m.Apply(context.Background(), []config.ClusterConfig{cfg}); err != nil { t.Fatal(err) }
	if !oldClient.closed { t.Fatal("old client was not closed") }
	if got, ok := m.Get("dev"); !ok || got != newClient { t.Fatal("new client not installed") }
}

func TestManagerKeepsOldClientWhenReplacementFails(t *testing.T) {
	oldClient := &fakeClient{}
	m := NewManager(fakeFactory{next: oldClient})
	cfg := config.ClusterConfig{ID: "dev", Name: "Dev", Brokers: []string{"localhost:9092"}}
	if err := m.Apply(context.Background(), []config.ClusterConfig{cfg}); err != nil { t.Fatal(err) }
	m.factory = fakeFactory{err: errors.New("connection refused")}
	if err := m.Apply(context.Background(), []config.ClusterConfig{cfg}); err == nil { t.Fatal("expected replacement error") }
	if got, ok := m.Get("dev"); !ok || got != oldClient || oldClient.closed { t.Fatal("old client was not preserved") }
}

func TestManagerRemovesMissingCluster(t *testing.T) {
	client := &fakeClient{}
	m := NewManager(fakeFactory{next: client})
	if err := m.Apply(context.Background(), []config.ClusterConfig{{ID: "dev"}}); err != nil { t.Fatal(err) }
	if err := m.Apply(context.Background(), nil); err != nil { t.Fatal(err) }
	if _, ok := m.Get("dev"); ok || !client.closed { t.Fatal("removed cluster remains active") }
}

func TestManagerUpsertKeepsOtherClusters(t *testing.T) {
	dev, test := &fakeClient{}, &fakeClient{}
	m := NewManager(fakeFactory{next: dev})
	if err := m.Upsert(context.Background(), config.ClusterConfig{ID: "dev"}); err != nil { t.Fatal(err) }
	m.factory = fakeFactory{next: test}
	if err := m.Upsert(context.Background(), config.ClusterConfig{ID: "test"}); err != nil { t.Fatal(err) }
	if _, ok := m.Get("dev"); !ok { t.Fatal("upsert removed existing cluster") }
	if got, ok := m.Get("test"); !ok || got != test { t.Fatal("new cluster not installed") }
}
