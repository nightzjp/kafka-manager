package topic

import (
	"context"
	"testing"
)

type fakeAdmin struct {
	topics  []Topic
	created CreateRequest
	deleted string
	configs []Config
	changes map[string]*string
}

func (f *fakeAdmin) List(context.Context) ([]Topic, error) { return f.topics, nil }
func (f *fakeAdmin) Create(_ context.Context, request CreateRequest) error {
	f.created = request
	return nil
}
func (f *fakeAdmin) Delete(_ context.Context, name string) error       { f.deleted = name; return nil }
func (f *fakeAdmin) AddPartitions(context.Context, string, int) error  { return nil }
func (f *fakeAdmin) Configs(context.Context, string) ([]Config, error) { return f.configs, nil }
func (f *fakeAdmin) AlterConfigs(_ context.Context, _ string, changes map[string]*string) error {
	f.changes = changes
	return nil
}

func TestServiceListsAndFiltersTopics(t *testing.T) {
	admin := &fakeAdmin{topics: []Topic{{Name: "orders", PartitionCount: 3}, {Name: "payments", PartitionCount: 2}}}
	result, total, err := NewService(admin).List(context.Background(), "pay", 1, 20)
	if err != nil || total != 1 || len(result) != 1 || result[0].Name != "payments" {
		t.Fatalf("List() = %+v, %d, %v", result, total, err)
	}
}

func TestServiceListsAndAltersTopicConfigs(t *testing.T) {
	value := "compact"
	admin := &fakeAdmin{configs: []Config{{Name: "cleanup.policy", Value: &value, Source: "DYNAMIC_TOPIC_CONFIG"}}}
	service := NewService(admin)
	configs, err := service.Configs(context.Background(), "orders")
	if err != nil || len(configs) != 1 || *configs[0].Value != "compact" {
		t.Fatalf("Configs() = %+v, %v", configs, err)
	}
	changes := map[string]*string{"cleanup.policy": &value, "retention.ms": nil}
	if err := service.AlterConfigs(context.Background(), "orders", changes); err != nil {
		t.Fatal(err)
	}
	if admin.changes["retention.ms"] != nil || *admin.changes["cleanup.policy"] != "compact" {
		t.Fatalf("changes = %+v", admin.changes)
	}
}

func TestServiceRejectsInvalidTopicConfigChanges(t *testing.T) {
	service := NewService(&fakeAdmin{})
	if err := service.AlterConfigs(context.Background(), "", map[string]*string{"x": nil}); err == nil {
		t.Fatal("accepted empty topic")
	}
	if err := service.AlterConfigs(context.Background(), "orders", map[string]*string{}); err == nil {
		t.Fatal("accepted empty changes")
	}
	if err := service.AlterConfigs(context.Background(), "orders", map[string]*string{"": nil}); err == nil {
		t.Fatal("accepted empty config name")
	}
}

func TestServiceRejectsInvalidCreate(t *testing.T) {
	service := NewService(&fakeAdmin{})
	for _, request := range []CreateRequest{{Name: "", Partitions: 1, ReplicationFactor: 1}, {Name: "x", Partitions: 0, ReplicationFactor: 1}, {Name: "x", Partitions: 1, ReplicationFactor: 0}} {
		if err := service.Create(context.Background(), request); err == nil {
			t.Fatalf("accepted invalid request %+v", request)
		}
	}
}

func TestTargetPartitionCountAddsIncrementToCurrentCount(t *testing.T) {
	if got := targetPartitionCount(6, 2); got != 8 {
		t.Fatalf("targetPartitionCount(6, 2) = %d, want 8", got)
	}
}
