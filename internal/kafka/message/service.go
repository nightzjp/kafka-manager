package message

import (
	"context"
	"fmt"
	"strings"
)

type Header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type Record struct {
	Topic     string   `json:"topic"`
	Partition int32    `json:"partition"`
	Offset    int64    `json:"offset"`
	Timestamp int64    `json:"timestamp"`
	Key       string   `json:"key"`
	Value     string   `json:"value"`
	Headers   []Header `json:"headers,omitempty"`
}
type Query struct {
	Topic             string
	Partition         int32
	Mode              string
	Offset, Timestamp int64
	Limit             int
}
type ProduceRequest struct {
	Topic     string   `json:"topic"`
	Partition int32    `json:"partition"`
	Key       string   `json:"key"`
	Value     string   `json:"value"`
	Headers   []Header `json:"headers,omitempty"`
}
type Backend interface {
	Fetch(context.Context, Query) ([]Record, error)
	Produce(context.Context, ProduceRequest) (Record, error)
	Stream(context.Context, Query, func(Record) error) error
}
type Service struct{ backend Backend }

func NewService(backend Backend) *Service { return &Service{backend: backend} }
func (s *Service) Query(ctx context.Context, q Query) ([]Record, error) {
	q, err := validateQuery(q)
	if err != nil {
		return nil, err
	}
	return s.backend.Fetch(ctx, q)
}
func validateQuery(q Query) (Query, error) {
	if strings.TrimSpace(q.Topic) == "" {
		return q, fmt.Errorf("topic is required")
	}
	if q.Partition < -1 {
		return q, fmt.Errorf("partition is invalid")
	}
	switch q.Mode {
	case "earliest", "latest", "live":
	case "offset":
		if q.Offset < 0 {
			return q, fmt.Errorf("offset must not be negative")
		}
	case "timestamp":
		if q.Timestamp <= 0 {
			return q, fmt.Errorf("timestamp is required")
		}
	default:
		return q, fmt.Errorf("unsupported query mode %q", q.Mode)
	}
	if q.Limit < 1 {
		q.Limit = 100
	}
	if q.Limit > 500 {
		q.Limit = 500
	}
	return q, nil
}
func (s *Service) Stream(ctx context.Context, q Query, send func(Record) error) error {
	q, err := validateQuery(q)
	if err != nil {
		return err
	}
	return s.backend.Stream(ctx, q, send)
}
func (s *Service) Produce(ctx context.Context, r ProduceRequest) (Record, error) {
	if strings.TrimSpace(r.Topic) == "" {
		return Record{}, fmt.Errorf("topic is required")
	}
	if len(r.Value) > 10*1024*1024 {
		return Record{}, fmt.Errorf("message exceeds 10 MiB")
	}
	return s.backend.Produce(ctx, r)
}
