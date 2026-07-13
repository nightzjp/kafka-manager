package topic

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/twmb/franz-go/pkg/kadm"
)

type Partition struct {
	ID, Leader                     int32
	Replicas, ISR, OfflineReplicas []int32
}
type Topic struct {
	Name                              string
	Internal                          bool
	PartitionCount, ReplicationFactor int
	UnderReplicated                   int
	Partitions                        []Partition
}
type CreateRequest struct {
	Name              string            `json:"name"`
	Partitions        int32             `json:"partitions"`
	ReplicationFactor int16             `json:"replicationFactor"`
	Configs           map[string]string `json:"configs,omitempty"`
}
type Config struct {
	Name      string  `json:"name"`
	Value     *string `json:"value"`
	Sensitive bool    `json:"sensitive"`
	Source    string  `json:"source"`
}

type Admin interface {
	List(context.Context) ([]Topic, error)
	Create(context.Context, CreateRequest) error
	Delete(context.Context, string) error
	AddPartitions(context.Context, string, int) error
	Configs(context.Context, string) ([]Config, error)
	AlterConfigs(context.Context, string, map[string]*string) error
}

type Service struct{ admin Admin }

func NewService(admin Admin) *Service { return &Service{admin: admin} }

func (s *Service) List(ctx context.Context, search string, page, pageSize int) ([]Topic, int, error) {
	items, err := s.admin.List(ctx)
	if err != nil {
		return nil, 0, err
	}
	search = strings.ToLower(strings.TrimSpace(search))
	filtered := items[:0]
	for _, item := range items {
		if search == "" || strings.Contains(strings.ToLower(item.Name), search) {
			filtered = append(filtered, item)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Name < filtered[j].Name })
	total := len(filtered)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []Topic{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return filtered[start:end], total, nil
}

func (s *Service) Create(ctx context.Context, request CreateRequest) error {
	if strings.TrimSpace(request.Name) == "" {
		return fmt.Errorf("topic name is required")
	}
	if request.Partitions < 1 {
		return fmt.Errorf("partitions must be greater than zero")
	}
	if request.ReplicationFactor < 1 {
		return fmt.Errorf("replicationFactor must be greater than zero")
	}
	return s.admin.Create(ctx, request)
}
func (s *Service) Delete(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("topic name is required")
	}
	return s.admin.Delete(ctx, name)
}
func (s *Service) AddPartitions(ctx context.Context, name string, count int) error {
	if count < 1 {
		return fmt.Errorf("partition increment must be positive")
	}
	return s.admin.AddPartitions(ctx, name, count)
}
func (s *Service) Configs(ctx context.Context, name string) ([]Config, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("topic name is required")
	}
	return s.admin.Configs(ctx, name)
}
func (s *Service) AlterConfigs(ctx context.Context, name string, changes map[string]*string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("topic name is required")
	}
	if len(changes) == 0 {
		return fmt.Errorf("at least one config change is required")
	}
	for key := range changes {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("config name is required")
		}
	}
	return s.admin.AlterConfigs(ctx, name, changes)
}

type KadmAdmin struct{ client *kadm.Client }

func NewKadmAdmin(client *kadm.Client) *KadmAdmin { return &KadmAdmin{client: client} }
func (a *KadmAdmin) List(ctx context.Context) ([]Topic, error) {
	details, err := a.client.ListTopics(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]Topic, 0, len(details))
	for _, detail := range details.Sorted() {
		item := Topic{Name: detail.Topic, Internal: detail.IsInternal, PartitionCount: len(detail.Partitions), ReplicationFactor: detail.Partitions.NumReplicas()}
		for _, p := range detail.Partitions.Sorted() {
			if len(p.ISR) < len(p.Replicas) {
				item.UnderReplicated++
			}
			item.Partitions = append(item.Partitions, Partition{ID: p.Partition, Leader: p.Leader, Replicas: p.Replicas, ISR: p.ISR, OfflineReplicas: p.OfflineReplicas})
		}
		items = append(items, item)
	}
	return items, nil
}
func (a *KadmAdmin) Create(ctx context.Context, request CreateRequest) error {
	configs := map[string]*string{}
	for k, v := range request.Configs {
		value := v
		configs[k] = &value
	}
	_, err := a.client.CreateTopic(ctx, request.Partitions, request.ReplicationFactor, configs, request.Name)
	return err
}
func (a *KadmAdmin) Delete(ctx context.Context, name string) error {
	_, err := a.client.DeleteTopic(ctx, name)
	return err
}
func (a *KadmAdmin) AddPartitions(ctx context.Context, name string, count int) error {
	details, err := a.client.ListTopics(ctx, name)
	if err != nil {
		return err
	}
	detail, ok := details[name]
	if !ok {
		return fmt.Errorf("topic not found")
	}
	targetCount := targetPartitionCount(len(detail.Partitions), count)
	responses, err := a.client.CreatePartitions(ctx, targetCount, name)
	if err != nil {
		return err
	}
	return responses.Error()
}

func targetPartitionCount(current, increment int) int { return current + increment }
func (a *KadmAdmin) Configs(ctx context.Context, name string) ([]Config, error) {
	resources, err := a.client.DescribeTopicConfigs(ctx, name)
	if err != nil {
		return nil, err
	}
	resource, err := resources.On(name, nil)
	if err != nil {
		return nil, err
	}
	if resource.Err != nil {
		return nil, resource.Err
	}
	items := make([]Config, 0, len(resource.Configs))
	for _, item := range resource.Configs {
		items = append(items, Config{Name: item.Key, Value: item.Value, Sensitive: item.Sensitive, Source: item.Source.String()})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}
func (a *KadmAdmin) AlterConfigs(ctx context.Context, name string, changes map[string]*string) error {
	configs := make([]kadm.AlterConfig, 0, len(changes))
	for key, value := range changes {
		op := kadm.SetConfig
		if value == nil {
			op = kadm.DeleteConfig
		}
		configs = append(configs, kadm.AlterConfig{Op: op, Name: key, Value: value})
	}
	sort.Slice(configs, func(i, j int) bool { return configs[i].Name < configs[j].Name })
	responses, err := a.client.AlterTopicConfigs(ctx, configs, name)
	if err != nil {
		return err
	}
	response, err := responses.On(name, nil)
	if err != nil {
		return err
	}
	return response.Err
}
