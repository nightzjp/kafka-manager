package message

import (
	"context"
	"testing"
)

type fakeBackend struct {
	query    Query
	produced ProduceRequest
	streamed Query
}

func (f *fakeBackend) Stream(_ context.Context, q Query, send func(Record) error) error {
	f.streamed = q
	return send(Record{Topic: q.Topic})
}

func (f *fakeBackend) Fetch(_ context.Context, q Query) (QueryResult, error) {
	f.query = q
	return QueryResult{Items: []Record{{Topic: q.Topic}}}, nil
}
func (f *fakeBackend) Produce(_ context.Context, r ProduceRequest) (Record, error) {
	f.produced = r
	return Record{Topic: r.Topic}, nil
}

func TestQueryAppliesSafeLimit(t *testing.T) {
	backend := &fakeBackend{}
	result, err := NewService(backend).Query(context.Background(), Query{Topic: "orders", Partition: 0, Mode: "latest", Limit: 5000})
	if err != nil || len(result.Items) != 1 {
		t.Fatal(err)
	}
	if backend.query.Limit != 500 || backend.query.ScanLimit != 500 {
		t.Fatalf("query=%+v want limit=scanLimit=500", backend.query)
	}
	_, err = NewService(backend).Query(context.Background(), Query{Topic: "orders", Partition: 0, Mode: "latest", Limit: 25, KeyFilter: "id"})
	if err != nil || backend.query.ScanLimit != 5000 {
		t.Fatalf("filtered query=%+v err=%v", backend.query, err)
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
func TestStreamValidatesAndUsesBoundedQuery(t *testing.T) {
	backend := &fakeBackend{}
	var received Record
	err := NewService(backend).Stream(context.Background(), Query{Topic: "orders", Partition: -1, Mode: "latest", Limit: 900}, func(record Record) error {
		received = record
		return nil
	})
	if err != nil || received.Topic != "orders" || backend.streamed.Limit != 500 {
		t.Fatalf("Stream() record=%+v query=%+v err=%v", received, backend.streamed, err)
	}
	if err := NewService(backend).Stream(context.Background(), Query{}, func(Record) error { return nil }); err == nil {
		t.Fatal("accepted invalid stream query")
	}
}
