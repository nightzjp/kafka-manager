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
	ScanLimit         int
	KeyFilter         string
	KeyOperator       string
	ValueFilter       string
	ValueOperator     string
	JSONFilters       []JSONFilter
}

type JSONFilter struct {
	Path     string `json:"path"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type QueryResult struct {
	Items              []Record `json:"items"`
	Scanned            int      `json:"scanned"`
	Matched            int      `json:"matched"`
	SkippedInvalidJSON int      `json:"skippedInvalidJson"`
	ResultLimited      bool     `json:"resultLimited"`
	ScanLimited        bool     `json:"scanLimited"`
}
type ProduceRequest struct {
	Topic     string   `json:"topic"`
	Partition int32    `json:"partition"`
	Key       string   `json:"key"`
	Value     string   `json:"value"`
	Headers   []Header `json:"headers,omitempty"`
}
type Backend interface {
	Fetch(context.Context, Query) (QueryResult, error)
	Produce(context.Context, ProduceRequest) (Record, error)
	Stream(context.Context, Query, func(Record) error) error
}
type Service struct{ backend Backend }

func NewService(backend Backend) *Service  { return &Service{backend: backend} }
func ValidateQuery(q Query) (Query, error) { return validateQuery(q) }
func (s *Service) Query(ctx context.Context, q Query) (QueryResult, error) {
	q, err := validateQuery(q)
	if err != nil {
		return QueryResult{}, err
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
	if q.ScanLimit > 50000 {
		return q, fmt.Errorf("scan limit must not exceed 50000")
	}
	if q.KeyFilter != "" {
		if q.KeyOperator == "" {
			q.KeyOperator = "contains"
		}
		if q.KeyOperator != "contains" && q.KeyOperator != "exact" && q.KeyOperator != "prefix" {
			return q, fmt.Errorf("unsupported key operator %q", q.KeyOperator)
		}
	}
	if q.ValueFilter != "" {
		if q.ValueOperator == "" {
			q.ValueOperator = "contains"
		}
		if q.ValueOperator != "contains" && q.ValueOperator != "exact" {
			return q, fmt.Errorf("unsupported value operator %q", q.ValueOperator)
		}
	}
	if len(q.JSONFilters) > 5 {
		return q, fmt.Errorf("at most 5 JSON filters are allowed")
	}
	for i := range q.JSONFilters {
		q.JSONFilters[i].Path = strings.TrimSpace(q.JSONFilters[i].Path)
		if q.JSONFilters[i].Path == "" {
			return q, fmt.Errorf("JSON filter path is required")
		}
		if q.JSONFilters[i].Operator == "" {
			q.JSONFilters[i].Operator = "eq"
		}
		switch q.JSONFilters[i].Operator {
		case "eq", "neq", "contains", "exists", "gt", "gte", "lt", "lte":
		default:
			return q, fmt.Errorf("unsupported JSON operator %q", q.JSONFilters[i].Operator)
		}
	}
	if hasContentFilters(q) {
		if q.ScanLimit < 1 {
			q.ScanLimit = 5000
		}
	} else {
		q.ScanLimit = q.Limit
	}
	return q, nil
}
func (s *Service) Stream(ctx context.Context, q Query, send func(Record) error) error {
	q, err := validateQuery(q)
	if err != nil {
		return err
	}
	return s.backend.Stream(ctx, q, func(record Record) error {
		matched, _ := matchRecord(record, q)
		if !matched {
			return nil
		}
		return send(record)
	})
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
