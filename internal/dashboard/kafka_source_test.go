package dashboard

import (
	"context"
	"errors"
	"testing"

	"github.com/nightzjp/kafka-manager/internal/config"
	"github.com/twmb/franz-go/pkg/kgo"
)

type reconnectingProvider struct {
	upserts int
	err     error
}

func (p *reconnectingProvider) Kafka(string) (*kgo.Client, bool) { return nil, false }
func (p *reconnectingProvider) Upsert(context.Context, config.ClusterConfig) error {
	p.upserts++
	return p.err
}

func TestKafkaSourceReconnectsMissingClient(t *testing.T) {
	provider := &reconnectingProvider{err: errors.New("broker unavailable")}
	source := KafkaSource{Clusters: provider}

	result := source.Snapshot(context.Background(), config.ClusterConfig{ID: "dev", Name: "开发环境"})
	if provider.upserts != 1 {
		t.Fatalf("upserts = %d, want 1", provider.upserts)
	}
	if result.Status != StatusOffline || result.Error == "" || result.Online {
		t.Fatalf("result = %+v", result)
	}
}

func TestRecordLagDistinguishesQueryFailureFromZero(t *testing.T) {
	failed := Snapshot{Status: StatusOnline, Online: true}
	recordLag(&failed, 0, errors.New("lag request timed out"))
	if failed.LagAvailable || failed.LagError == "" || failed.TotalLag != 0 {
		t.Fatalf("failed lag result = %+v", failed)
	}

	zero := Snapshot{Status: StatusOnline, Online: true}
	recordLag(&zero, 0, nil)
	if !zero.LagAvailable || zero.LagError != "" || zero.TotalLag != 0 {
		t.Fatalf("zero lag result = %+v", zero)
	}
}
