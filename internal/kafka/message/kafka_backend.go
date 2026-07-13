package message

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nightzjp/kafka-manager/internal/cluster"
	"github.com/nightzjp/kafka-manager/internal/config"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

type KafkaBackend struct {
	cfg      config.ClusterConfig
	producer *kgo.Client
}

func NewKafkaBackend(cfg config.ClusterConfig, producer *kgo.Client) *KafkaBackend {
	return &KafkaBackend{cfg: cfg, producer: producer}
}

func (b *KafkaBackend) Fetch(ctx context.Context, q Query) (QueryResult, error) {
	admin := kadm.NewClient(b.producer)
	partitions := []int32{q.Partition}
	if q.Partition == -1 {
		details, err := admin.ListTopics(ctx, q.Topic)
		if err != nil {
			return QueryResult{}, err
		}
		detail, ok := details[q.Topic]
		if !ok {
			return QueryResult{}, fmt.Errorf("topic not found")
		}
		partitions = detail.Partitions.Numbers()
	}
	assignment := map[string]map[int32]kgo.Offset{q.Topic: {}}
	for _, partition := range partitions {
		offset, err := b.resolveOffset(ctx, admin, q, partition)
		if err != nil {
			return QueryResult{}, err
		}
		assignment[q.Topic][partition] = kgo.NewOffset().At(offset)
	}
	opts, err := cluster.Options(b.cfg)
	if err != nil {
		return QueryResult{}, err
	}
	opts = append(opts, kgo.ConsumePartitions(assignment), kgo.FetchMaxBytes(20*1024*1024))
	consumer, err := kgo.NewClient(opts...)
	if err != nil {
		return QueryResult{}, err
	}
	defer consumer.Close()
	collector := newRecordCollector(q)
	for !collector.done() {
		fetchCtx, cancel := context.WithTimeout(ctx, 1200*time.Millisecond)
		batch := consumer.PollRecords(fetchCtx, collector.remainingScan())
		pollCtxErr := fetchCtx.Err()
		cancel()
		if errs := batch.Errors(); len(errs) > 0 {
			for _, fetchErr := range errs {
				if err := pollError(fetchErr.Err, pollCtxErr, ctx.Err()); err != nil {
					return QueryResult{}, err
				}
			}
		}
		if batch.Empty() {
			break
		}
		batch.EachRecord(func(record *kgo.Record) {
			if collector.done() {
				return
			}
			headers := make([]Header, 0, len(record.Headers))
			for _, h := range record.Headers {
				headers = append(headers, Header{Key: h.Key, Value: string(h.Value)})
			}
			collector.add(Record{Topic: record.Topic, Partition: record.Partition, Offset: record.Offset, Timestamp: record.Timestamp.UnixMilli(), Key: string(record.Key), Value: string(record.Value), Headers: headers})
		})
	}
	return collector.result(), nil
}

func pollError(fetchErr, pollContextErr, requestContextErr error) error {
	if requestContextErr != nil {
		return requestContextErr
	}
	if errors.Is(pollContextErr, context.DeadlineExceeded) && errors.Is(fetchErr, context.DeadlineExceeded) {
		return nil
	}
	return fetchErr
}
func (b *KafkaBackend) resolveOffset(ctx context.Context, admin *kadm.Client, q Query, partition int32) (int64, error) {
	if q.Mode == "offset" {
		return q.Offset, nil
	}
	var listed kadm.ListedOffsets
	var err error
	switch q.Mode {
	case "earliest":
		listed, err = admin.ListStartOffsets(ctx, q.Topic)
	case "latest", "live":
		listed, err = admin.ListEndOffsets(ctx, q.Topic)
	case "timestamp":
		listed, err = admin.ListOffsetsAfterMilli(ctx, q.Timestamp, q.Topic)
	}
	if err != nil {
		return 0, err
	}
	offset, ok := listed.Lookup(q.Topic, partition)
	if !ok || offset.Err != nil {
		return 0, fmt.Errorf("offset unavailable for partition %d", partition)
	}
	at := offset.Offset
	if q.Mode == "latest" {
		at -= int64(q.ScanLimit)
		if at < 0 {
			at = 0
		}
	}
	return at, nil
}
func (b *KafkaBackend) Stream(ctx context.Context, q Query, send func(Record) error) error {
	q.Mode = "live"
	admin := kadm.NewClient(b.producer)
	partitions := []int32{q.Partition}
	if q.Partition == -1 {
		details, err := admin.ListTopics(ctx, q.Topic)
		if err != nil {
			return err
		}
		detail, ok := details[q.Topic]
		if !ok {
			return fmt.Errorf("topic not found")
		}
		partitions = detail.Partitions.Numbers()
	}
	assignment := map[string]map[int32]kgo.Offset{q.Topic: {}}
	for _, partition := range partitions {
		offset, err := b.resolveOffset(ctx, admin, q, partition)
		if err != nil {
			return err
		}
		assignment[q.Topic][partition] = kgo.NewOffset().At(offset)
	}
	opts, err := cluster.Options(b.cfg)
	if err != nil {
		return err
	}
	opts = append(opts, kgo.ConsumePartitions(assignment), kgo.FetchMaxBytes(20*1024*1024))
	consumer, err := kgo.NewClient(opts...)
	if err != nil {
		return err
	}
	defer consumer.Close()
	for {
		batch := consumer.PollRecords(ctx, 100)
		if ctx.Err() != nil {
			return nil
		}
		if errs := batch.Errors(); len(errs) > 0 {
			return errs[0].Err
		}
		var sendErr error
		batch.EachRecord(func(record *kgo.Record) {
			if sendErr != nil {
				return
			}
			headers := make([]Header, 0, len(record.Headers))
			for _, h := range record.Headers {
				headers = append(headers, Header{Key: h.Key, Value: string(h.Value)})
			}
			sendErr = send(Record{Topic: record.Topic, Partition: record.Partition, Offset: record.Offset, Timestamp: record.Timestamp.UnixMilli(), Key: string(record.Key), Value: string(record.Value), Headers: headers})
		})
		if sendErr != nil {
			return sendErr
		}
	}
}
func (b *KafkaBackend) Produce(ctx context.Context, r ProduceRequest) (Record, error) {
	headers := make([]kgo.RecordHeader, 0, len(r.Headers))
	for _, h := range r.Headers {
		headers = append(headers, kgo.RecordHeader{Key: h.Key, Value: []byte(h.Value)})
	}
	record := &kgo.Record{Topic: r.Topic, Key: []byte(r.Key), Value: []byte(r.Value), Headers: headers}
	if r.Partition >= 0 {
		record.Partition = r.Partition
	}
	producer := b.producer
	if r.Partition >= 0 {
		opts, err := cluster.Options(b.cfg)
		if err != nil {
			return Record{}, err
		}
		opts = append(opts, kgo.RecordPartitioner(kgo.ManualPartitioner()))
		manual, err := kgo.NewClient(opts...)
		if err != nil {
			return Record{}, err
		}
		defer manual.Close()
		producer = manual
	}
	result, err := producer.ProduceSync(ctx, record).First()
	if err != nil {
		return Record{}, err
	}
	return Record{Topic: result.Topic, Partition: result.Partition, Offset: result.Offset, Timestamp: result.Timestamp.UnixMilli(), Key: r.Key, Value: r.Value, Headers: r.Headers}, nil
}
