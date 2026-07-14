package dashboard

import (
	"context"
	"time"

	"github.com/nightzjp/kafka-manager/internal/config"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

type ClusterProvider interface {
	Kafka(string) (*kgo.Client, bool)
	Upsert(context.Context, config.ClusterConfig) error
}

type KafkaSource struct {
	Clusters ClusterProvider
	Timeout  time.Duration
}

func (s KafkaSource) Snapshot(parent context.Context, cfg config.ClusterConfig) Snapshot {
	result := Snapshot{Status: StatusOffline}
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = 4 * time.Second
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	client, ok := s.Clusters.Kafka(cfg.ID)
	if !ok {
		if err := s.Clusters.Upsert(ctx, cfg); err != nil {
			result.Error = err.Error()
			return result
		}
		client, ok = s.Clusters.Kafka(cfg.ID)
		if !ok {
			result.Error = "集群客户端不可用"
			return result
		}
	}

	start := time.Now()
	admin := kadm.NewClient(client)
	brokers, err := admin.ListBrokers(ctx)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	topics, err := admin.ListTopics(ctx)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	groups, err := admin.ListGroups(ctx)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Status = StatusOnline
	result.Online = true
	result.LatencyMS = time.Since(start).Milliseconds()
	result.Brokers = len(brokers)
	result.Topics = len(topics)
	result.ConsumerGroups = len(groups)
	for _, topic := range topics {
		result.Partitions += len(topic.Partitions)
		for _, partition := range topic.Partitions {
			if len(partition.ISR) < len(partition.Replicas) {
				result.UnderReplicated++
			}
		}
	}
	if len(groups) == 0 {
		recordLag(&result, 0, nil)
		return result
	}
	lags, lagErr := admin.Lag(ctx, groups.Groups()...)
	var total int64
	if lagErr == nil {
		for _, group := range lags {
			total += group.Lag.Total()
		}
	}
	recordLag(&result, total, lagErr)
	return result
}

func recordLag(result *Snapshot, total int64, err error) {
	if err != nil {
		result.LagAvailable = false
		result.LagError = err.Error()
		return
	}
	result.LagAvailable = true
	result.LagError = ""
	result.TotalLag = total
}
