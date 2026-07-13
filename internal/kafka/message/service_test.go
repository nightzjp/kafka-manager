package message

import (
	"context"
	"testing"
)

type fakeBackend struct {
	query    Query
	produced ProduceRequest
}

func (f *fakeBackend) Fetch(_ context.Context, q Query) ([]Record, error) {
	f.query = q
	return []Record{{Topic: q.Topic}}, nil
}
func (f *fakeBackend) Produce(_ context.Context, r ProduceRequest) (Record, error) {
	f.produced = r
	return Record{Topic: r.Topic}, nil
}

func TestQueryAppliesSafeLimit(t *testing.T) {
	backend := &fakeBackend{}
	records, err := NewService(backend).Query(context.Background(), Query{Topic: "orders", Partition: 0, Mode: "latest", Limit: 5000})
	if err != nil || len(records) != 1 {
		t.Fatal(err)
	}
	if backend.query.Limit != 500 {
		t.Fatalf("limit=%d want 500", backend.query.Limit)
	}
}
func TestQueryRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeBackend{})
	for _, q := range []Query{{Topic: "", Mode: "latest"}, {Topic: "x", Partition: -2, Mode: "latest"}, {Topic: "x", Partition: 0, Mode: "bad"}, {Topic: "x", Partition: 0, Mode: "offset", Offset: -1}} {
		if _, err := service.Query(context.Background(), q); err == nil {
			t.Fatalf("accepted %+v", q)
		}
	}
}
func TestProduceRequiresTopic(t *testing.T) {
	if _, err := NewService(&fakeBackend{}).Produce(context.Background(), ProduceRequest{}); err == nil {
		t.Fatal("empty topic accepted")
	}
}
