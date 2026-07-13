package topic

import (
	"context"
	"testing"
)

type fakeAdmin struct {
	topics  []Topic
	created CreateRequest
	deleted string
}

func (f *fakeAdmin) List(context.Context) ([]Topic, error) { return f.topics, nil }
func (f *fakeAdmin) Create(_ context.Context, request CreateRequest) error {
	f.created = request
	return nil
}
func (f *fakeAdmin) Delete(_ context.Context, name string) error      { f.deleted = name; return nil }
func (f *fakeAdmin) AddPartitions(context.Context, string, int) error { return nil }

func TestServiceListsAndFiltersTopics(t *testing.T) {
	admin := &fakeAdmin{topics: []Topic{{Name: "orders", PartitionCount: 3}, {Name: "payments", PartitionCount: 2}}}
	result, total, err := NewService(admin).List(context.Background(), "pay", 1, 20)
	if err != nil || total != 1 || len(result) != 1 || result[0].Name != "payments" {
		t.Fatalf("List() = %+v, %d, %v", result, total, err)
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
