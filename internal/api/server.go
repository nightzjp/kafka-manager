package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nightzjp/kafka-manager/internal/audit"
	"github.com/nightzjp/kafka-manager/internal/auth"
	"github.com/nightzjp/kafka-manager/internal/cluster"
	"github.com/nightzjp/kafka-manager/internal/config"
	"github.com/nightzjp/kafka-manager/internal/dashboard"
	consumerService "github.com/nightzjp/kafka-manager/internal/kafka/consumer"
	messageService "github.com/nightzjp/kafka-manager/internal/kafka/message"
	topicService "github.com/nightzjp/kafka-manager/internal/kafka/topic"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Server struct {
	mu          sync.RWMutex
	configTxn   sync.RWMutex
	cfg         config.Config
	store       *config.Store
	clusters    *cluster.Manager
	secret      []byte
	auditWriter *audit.Writer
	auditDir    string
	monitor     *dashboard.Sampler
	handler     http.Handler
}

func NewServer(cfg config.Config, store *config.Store, clusters *cluster.Manager, sessionKey []byte) *Server {
	clusters.SetDesired(cfg.Clusters)
	monitor := dashboard.NewSampler(cfg.Clusters, dashboardOptions(cfg), dashboard.KafkaSource{Clusters: clusters})
	s := &Server{cfg: cfg, store: store, clusters: clusters, secret: append([]byte(nil), sessionKey...), monitor: monitor}
	mux := http.NewServeMux()
	authHandler := NewAuthHandler(cfg.Server.Username, cfg.Server.Password, cfg.Server.PasswordHash, auth.NewSessionManager(sessionKey, time.Duration(cfg.Server.SessionHours)*time.Hour))
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.Handle("POST /api/v1/auth/login", authHandler)
	mux.Handle("POST /api/v1/auth/logout", authHandler)
	mux.Handle("GET /api/v1/auth/me", authHandler)
	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/v1/clusters", s.listClusters)
	protected.HandleFunc("GET /api/v1/dashboard", s.dashboard)
	protected.HandleFunc("GET /api/v1/config", s.getConfig)
	protected.HandleFunc("PUT /api/v1/config", s.putConfig)
	protected.HandleFunc("GET /api/v1/config/backups", s.listConfigBackups)
	protected.HandleFunc("POST /api/v1/config/backups/{backup...}", s.restoreConfigBackup)
	protected.HandleFunc("GET /api/v1/audit", s.listAudit)
	protected.HandleFunc("GET /api/v1/clusters/{cluster}/topics", s.listTopics)
	protected.HandleFunc("POST /api/v1/clusters/{cluster}/topics", s.createTopic)
	protected.HandleFunc("DELETE /api/v1/clusters/{cluster}/topics/{topic}", s.deleteTopic)
	protected.HandleFunc("POST /api/v1/clusters/{cluster}/topics/{topic}/partitions", s.addPartitions)
	protected.HandleFunc("GET /api/v1/clusters/{cluster}/topics/{topic}/configs", s.listTopicConfigs)
	protected.HandleFunc("PUT /api/v1/clusters/{cluster}/topics/{topic}/configs", s.alterTopicConfigs)
	protected.HandleFunc("GET /api/v1/clusters/{cluster}/messages", s.listMessages)
	protected.HandleFunc("GET /api/v1/clusters/{cluster}/messages/stream", s.streamMessages)
	protected.HandleFunc("POST /api/v1/clusters/{cluster}/messages", s.produceMessage)
	protected.HandleFunc("GET /api/v1/clusters/{cluster}/consumer-groups", s.listConsumerGroups)
	protected.HandleFunc("POST /api/v1/clusters/{cluster}/consumer-groups/{group}/reset", s.resetConsumerGroup)
	protected.HandleFunc("DELETE /api/v1/clusters/{cluster}/consumer-groups/{group}", s.deleteConsumerGroup)
	mux.Handle("/api/v1/", auth.Middleware(auth.NewSessionManager(sessionKey, time.Duration(cfg.Server.SessionHours)*time.Hour), sessionCookieName, protected))
	s.handler = mux
	return s
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.handler.ServeHTTP(w, r) }
func (s *Server) UpdateConfig(cfg config.Config) {
	s.configTxn.Lock()
	defer s.configTxn.Unlock()
	s.updateConfig(cfg)
}
func (s *Server) updateConfig(cfg config.Config) {
	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()
	s.monitor.Update(cfg.Clusters, dashboardOptions(cfg))
}
func (s *Server) RunDashboard(ctx context.Context) { s.monitor.Run(ctx) }
func (s *Server) SetAudit(writer *audit.Writer, directory string) {
	s.mu.Lock()
	s.auditWriter = writer
	s.auditDir = directory
	s.mu.Unlock()
}
func (s *Server) getConfig(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	cfg := s.cfg
	s.mu.RUnlock()
	cfg.Server.PasswordHash = ""
	cfg.Server.Password = ""
	cfg.Clusters = append([]config.ClusterConfig(nil), cfg.Clusters...)
	for i := range cfg.Clusters {
		cfg.Clusters[i].Security.Password = ""
	}
	writeJSON(w, 200, cfg)
}
func (s *Server) putConfig(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeError(w, 503, "config_read_only", "配置文件不可写")
		return
	}
	var candidate config.Config
	if !decodeJSON(w, r, &candidate) {
		return
	}
	s.configTxn.Lock()
	defer s.configTxn.Unlock()
	s.mu.RLock()
	current := s.cfg
	s.mu.RUnlock()
	if candidate.Server.PasswordHash == "" {
		candidate.Server.PasswordHash = current.Server.PasswordHash
	}
	if candidate.Server.Password == "" {
		candidate.Server.Password = current.Server.Password
	}
	for i := range candidate.Clusters {
		if candidate.Clusters[i].Security.Password == "" {
			for _, old := range current.Clusters {
				if old.ID == candidate.Clusters[i].ID {
					candidate.Clusters[i].Security.Password = old.Security.Password
					break
				}
			}
		}
	}
	if err := candidate.Validate(); err != nil {
		writeError(w, 400, "invalid_config", err.Error())
		return
	}
	data, err := config.Marshal(candidate)
	if err != nil {
		writeError(w, 500, "encode_config", err.Error())
		return
	}
	connectionsChanged, err := s.applyClustersIfChanged(r.Context(), candidate.Clusters)
	if err != nil {
		writeError(w, 400, "connection_failed", err.Error())
		return
	}
	if _, err := s.store.Save(data); err != nil {
		if connectionsChanged {
			if rollbackErr := s.rollbackClusters(current.Clusters); rollbackErr != nil {
				writeError(w, 500, "save_config", fmt.Sprintf("%v; restore previous cluster connections: %v", err, rollbackErr))
				return
			}
		}
		writeError(w, 500, "save_config", err.Error())
		return
	}
	s.updateConfig(candidate)
	s.recordAudit(r, "config.update", "config.yaml", nil)
	w.WriteHeader(204)
}
func (s *Server) listConfigBackups(w http.ResponseWriter, _ *http.Request) {
	if s.store == nil {
		writeError(w, 503, "config_read_only", "配置文件不可写")
		return
	}
	items, err := s.store.ListBackups()
	if err != nil {
		writeError(w, 500, "backup_list", err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (s *Server) restoreConfigBackup(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeError(w, 503, "config_read_only", "配置文件不可写")
		return
	}
	data, persisted, err := s.store.LoadBackup(r.PathValue("backup"))
	if err != nil {
		writeError(w, 400, "backup_restore", err.Error())
		return
	}
	runtimeCfg, err := config.Runtime(persisted, s.secret)
	if err != nil {
		writeError(w, 400, "backup_decrypt", err.Error())
		return
	}
	s.configTxn.Lock()
	defer s.configTxn.Unlock()
	s.mu.RLock()
	current := s.cfg
	s.mu.RUnlock()
	connectionsChanged, err := s.applyClustersIfChanged(r.Context(), runtimeCfg.Clusters)
	if err != nil {
		writeError(w, 400, "connection_failed", err.Error())
		return
	}
	if _, err = s.store.Save(data); err != nil {
		if connectionsChanged {
			if rollbackErr := s.rollbackClusters(current.Clusters); rollbackErr != nil {
				writeError(w, 500, "backup_restore", fmt.Sprintf("%v; restore previous cluster connections: %v", err, rollbackErr))
				return
			}
		}
		writeError(w, 500, "backup_restore", err.Error())
		return
	}
	s.updateConfig(runtimeCfg)
	s.recordAudit(r, "config.restore", r.PathValue("backup"), nil)
	w.WriteHeader(204)
}

func (s *Server) applyClustersIfChanged(ctx context.Context, clusters []config.ClusterConfig) (bool, error) {
	if s.clusters.Matches(clusters) {
		return false, nil
	}
	if err := s.clusters.Apply(ctx, clusters); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Server) rollbackClusters(clusters []config.ClusterConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.clusters.Apply(ctx, clusters); err != nil {
		s.clusters.SetDesired(clusters)
		return err
	}
	return nil
}

// ReconcileConfig atomically switches the Kafka clients and the configuration
// observed by request handlers during an external file hot reload.
func (s *Server) ReconcileConfig(ctx context.Context, cfg config.Config) error {
	s.configTxn.Lock()
	defer s.configTxn.Unlock()
	if _, err := s.applyClustersIfChanged(ctx, cfg.Clusters); err != nil {
		return err
	}
	s.updateConfig(cfg)
	return nil
}

func (s *Server) clusterConfig(id string) (config.ClusterConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, cfg := range s.cfg.Clusters {
		if cfg.ID == id {
			return cfg, true
		}
	}
	return config.ClusterConfig{}, false
}
func (s *Server) kafka(id string) (*kadm.Client, config.ClusterConfig, error) {
	s.configTxn.RLock()
	defer s.configTxn.RUnlock()
	return s.kafkaLocked(id)
}

func (s *Server) kafkaLocked(id string) (*kadm.Client, config.ClusterConfig, error) {
	client, cfg, err := s.kafkaClientLocked(id)
	if err != nil {
		return nil, cfg, err
	}
	return kadm.NewClient(client), cfg, nil
}

func (s *Server) kafkaClient(id string) (*kgo.Client, config.ClusterConfig, error) {
	s.configTxn.RLock()
	defer s.configTxn.RUnlock()
	return s.kafkaClientLocked(id)
}

func (s *Server) kafkaClientLocked(id string) (*kgo.Client, config.ClusterConfig, error) {
	cfg, ok := s.clusterConfig(id)
	if !ok {
		return nil, cfg, fmt.Errorf("cluster %q not found", id)
	}
	client, ok := s.clusters.Kafka(id)
	if !ok {
		return nil, cfg, fmt.Errorf("cluster %q is offline", id)
	}
	return client, cfg, nil
}

func (s *Server) writableKafka(w http.ResponseWriter, r *http.Request, action, resource string) (*kadm.Client, config.ClusterConfig, bool) {
	s.configTxn.RLock()
	defer s.configTxn.RUnlock()
	cfg, exists := s.clusterConfig(r.PathValue("cluster"))
	if !exists {
		writeError(w, http.StatusServiceUnavailable, "cluster_unavailable", fmt.Sprintf("cluster %q not found", r.PathValue("cluster")))
		return nil, cfg, false
	}
	if cfg.ReadOnly {
		operationErr := fmt.Errorf("集群 %s 已启用只读模式，禁止执行写操作", cfg.Name)
		s.recordAudit(r, action, resource, operationErr)
		writeError(w, http.StatusForbidden, "cluster_read_only", operationErr.Error())
		return nil, cfg, false
	}
	admin, cfg, err := s.kafkaLocked(r.PathValue("cluster"))
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "cluster_unavailable", err.Error())
		return nil, cfg, false
	}
	return admin, cfg, true
}

func (s *Server) writableKafkaClient(w http.ResponseWriter, r *http.Request, action, resource string) (*kgo.Client, config.ClusterConfig, bool) {
	s.configTxn.RLock()
	defer s.configTxn.RUnlock()
	cfg, exists := s.clusterConfig(r.PathValue("cluster"))
	if !exists {
		writeError(w, http.StatusServiceUnavailable, "cluster_unavailable", fmt.Sprintf("cluster %q not found", r.PathValue("cluster")))
		return nil, cfg, false
	}
	if cfg.ReadOnly {
		operationErr := fmt.Errorf("集群 %s 已启用只读模式，禁止执行写操作", cfg.Name)
		s.recordAudit(r, action, resource, operationErr)
		writeError(w, http.StatusForbidden, "cluster_read_only", operationErr.Error())
		return nil, cfg, false
	}
	client, cfg, err := s.kafkaClientLocked(r.PathValue("cluster"))
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "cluster_unavailable", err.Error())
		return nil, cfg, false
	}
	return client, cfg, true
}

func (s *Server) listClusters(w http.ResponseWriter, _ *http.Request) {
	items, _ := s.monitor.Read()
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (s *Server) dashboard(w http.ResponseWriter, _ *http.Request) {
	items, history := s.monitor.Read()
	writeJSON(w, 200, map[string]any{"items": items, "history": history})
}

func dashboardOptions(cfg config.Config) dashboard.Options {
	return dashboard.Options{
		Interval:      time.Duration(cfg.Dashboard.SampleIntervalSeconds) * time.Second,
		HistoryPoints: cfg.Dashboard.HistoryPoints,
		MaxConcurrent: 4,
	}
}

func (s *Server) listTopics(w http.ResponseWriter, r *http.Request) {
	admin, _, err := s.kafka(r.PathValue("cluster"))
	if err != nil {
		writeError(w, 503, "cluster_unavailable", err.Error())
		return
	}
	page := intQuery(r, "page", 1)
	size := intQuery(r, "pageSize", 50)
	items, total, err := topicService.NewService(topicService.NewKadmAdmin(admin)).List(r.Context(), r.URL.Query().Get("search"), page, size)
	if err != nil {
		writeKafkaError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size})
}
func (s *Server) createTopic(w http.ResponseWriter, r *http.Request) {
	admin, _, ok := s.writableKafka(w, r, "topic.create", "")
	if !ok {
		return
	}
	var request topicService.CreateRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	err := topicService.NewService(topicService.NewKadmAdmin(admin)).Create(r.Context(), request)
	s.recordAudit(r, "topic.create", request.Name, err)
	if err != nil {
		writeKafkaError(w, err)
		return
	}
	writeJSON(w, 201, map[string]string{"status": "created"})
}
func (s *Server) deleteTopic(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("topic")
	admin, _, ok := s.writableKafka(w, r, "topic.delete", name)
	if !ok {
		return
	}
	err := topicService.NewService(topicService.NewKadmAdmin(admin)).Delete(r.Context(), name)
	s.recordAudit(r, "topic.delete", name, err)
	if err != nil {
		writeKafkaError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) addPartitions(w http.ResponseWriter, r *http.Request) {
	admin, _, ok := s.writableKafka(w, r, "topic.partitions.add", r.PathValue("topic"))
	if !ok {
		return
	}
	var request struct {
		Count int `json:"count"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	err := topicService.NewService(topicService.NewKadmAdmin(admin)).AddPartitions(r.Context(), r.PathValue("topic"), request.Count)
	s.recordAudit(r, "topic.partitions.add", r.PathValue("topic"), err)
	if err != nil {
		writeKafkaError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) listTopicConfigs(w http.ResponseWriter, r *http.Request) {
	admin, _, err := s.kafka(r.PathValue("cluster"))
	if err != nil {
		writeError(w, 503, "cluster_unavailable", err.Error())
		return
	}
	items, err := topicService.NewService(topicService.NewKadmAdmin(admin)).Configs(r.Context(), r.PathValue("topic"))
	if err != nil {
		writeKafkaError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (s *Server) alterTopicConfigs(w http.ResponseWriter, r *http.Request) {
	admin, _, ok := s.writableKafka(w, r, "topic.config.alter", r.PathValue("topic"))
	if !ok {
		return
	}
	var request struct {
		Configs map[string]*string `json:"configs"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	err := topicService.NewService(topicService.NewKadmAdmin(admin)).AlterConfigs(r.Context(), r.PathValue("topic"), request.Configs)
	s.recordAudit(r, "topic.config.alter", r.PathValue("topic"), err)
	if err != nil {
		writeKafkaError(w, err)
		return
	}
	w.WriteHeader(204)
}

func (s *Server) listMessages(w http.ResponseWriter, r *http.Request) {
	q, err := messageQuery(r, "")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_message_filter", err.Error())
		return
	}
	client, cfg, err := s.kafkaClient(r.PathValue("cluster"))
	if err != nil {
		writeError(w, 503, "cluster_unavailable", err.Error())
		return
	}
	result, err := messageService.NewService(messageService.NewKafkaBackend(cfg, client)).Query(r.Context(), q)
	if err != nil {
		writeKafkaError(w, err)
		return
	}
	writeJSON(w, 200, result)
}
func (s *Server) streamMessages(w http.ResponseWriter, r *http.Request) {
	q, err := messageQuery(r, "live")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_message_filter", err.Error())
		return
	}
	q, err = messageService.ValidateQuery(q)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_message_filter", err.Error())
		return
	}
	client, cfg, err := s.kafkaClient(r.PathValue("cluster"))
	if err != nil {
		writeError(w, 503, "cluster_unavailable", err.Error())
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, 500, "stream_unsupported", "服务器不支持消息流")
		return
	}
	q.Limit = 500
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher.Flush()
	err = messageService.NewService(messageService.NewKafkaBackend(cfg, client)).Stream(r.Context(), q, func(record messageService.Record) error {
		data, encodeErr := json.Marshal(record)
		if encodeErr != nil {
			return encodeErr
		}
		if _, writeErr := fmt.Fprintf(w, "data: %s\n\n", data); writeErr != nil {
			return writeErr
		}
		flusher.Flush()
		return nil
	})
	if err != nil && r.Context().Err() == nil {
		data, _ := json.Marshal(map[string]string{"error": err.Error()})
		_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
		flusher.Flush()
	}
}

func messageQuery(r *http.Request, mode string) (messageService.Query, error) {
	if mode == "" {
		mode = r.URL.Query().Get("mode")
	}
	query := messageService.Query{
		Topic:         r.URL.Query().Get("topic"),
		Partition:     int32(intQuery(r, "partition", -1)),
		Mode:          mode,
		Offset:        int64Query(r, "offset", 0),
		Timestamp:     int64Query(r, "timestamp", 0),
		Limit:         intQuery(r, "limit", 100),
		ScanLimit:     intQuery(r, "scanLimit", 0),
		KeyFilter:     r.URL.Query().Get("keyFilter"),
		KeyOperator:   r.URL.Query().Get("keyOperator"),
		ValueFilter:   r.URL.Query().Get("valueFilter"),
		ValueOperator: r.URL.Query().Get("valueOperator"),
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("jsonFilters")); raw != "" {
		if err := json.Unmarshal([]byte(raw), &query.JSONFilters); err != nil {
			return query, fmt.Errorf("JSON filters are invalid: %w", err)
		}
	}
	return query, nil
}
func (s *Server) produceMessage(w http.ResponseWriter, r *http.Request) {
	client, cfg, ok := s.writableKafkaClient(w, r, "message.produce", "")
	if !ok {
		return
	}
	var request messageService.ProduceRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	record, err := messageService.NewService(messageService.NewKafkaBackend(cfg, client)).Produce(r.Context(), request)
	s.recordAudit(r, "message.produce", request.Topic, err)
	if err != nil {
		writeKafkaError(w, err)
		return
	}
	writeJSON(w, 201, record)
}
func (s *Server) listConsumerGroups(w http.ResponseWriter, r *http.Request) {
	admin, _, err := s.kafka(r.PathValue("cluster"))
	if err != nil {
		writeError(w, 503, "cluster_unavailable", err.Error())
		return
	}
	items, err := consumerService.NewService(consumerService.NewKadmBackend(admin)).List(r.Context(), r.URL.Query().Get("search"))
	if err != nil {
		writeKafkaError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (s *Server) resetConsumerGroup(w http.ResponseWriter, r *http.Request) {
	admin, _, ok := s.writableKafka(w, r, "consumer.offset.reset", r.PathValue("group"))
	if !ok {
		return
	}
	var request consumerService.ResetRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	request.Group = r.PathValue("group")
	err := consumerService.NewService(consumerService.NewKadmBackend(admin)).Reset(r.Context(), request)
	s.recordAudit(r, "consumer.offset.reset", request.Group, err)
	if err != nil {
		writeKafkaError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) deleteConsumerGroup(w http.ResponseWriter, r *http.Request) {
	admin, _, ok := s.writableKafka(w, r, "consumer.group.delete", r.PathValue("group"))
	if !ok {
		return
	}
	err := consumerService.NewService(consumerService.NewKadmBackend(admin)).Delete(r.Context(), r.PathValue("group"))
	s.recordAudit(r, "consumer.group.delete", r.PathValue("group"), err)
	if err != nil {
		writeKafkaError(w, err)
		return
	}
	w.WriteHeader(204)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 11*1024*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, 400, "invalid_request", "请求格式不正确: "+err.Error())
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeKafkaError(w http.ResponseWriter, err error) {
	writeError(w, 400, "kafka_error", err.Error())
}
func intQuery(r *http.Request, name string, fallback int) int {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
func int64Query(r *http.Request, name string, fallback int64) int64 {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
func (s *Server) recordAudit(r *http.Request, action, resource string, operationErr error) {
	s.mu.RLock()
	writer := s.auditWriter
	s.mu.RUnlock()
	if writer == nil {
		return
	}
	username, _ := auth.Username(r.Context())
	result := "success"
	message := ""
	if operationErr != nil {
		result = "failed"
		message = operationErr.Error()
	}
	_ = writer.Write(r.Context(), audit.Entry{Username: username, ClientIP: r.RemoteAddr, ClusterID: r.PathValue("cluster"), Action: action, Resource: resource, Result: result, Error: message})
}
func (s *Server) listAudit(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	directory := s.auditDir
	s.mu.RUnlock()
	items, err := audit.Query(directory, audit.Filter{From: r.URL.Query().Get("from"), To: r.URL.Query().Get("to"), ClusterID: r.URL.Query().Get("cluster"), Action: r.URL.Query().Get("action"), Result: r.URL.Query().Get("result"), Limit: intQuery(r, "limit", 100)})
	if err != nil {
		writeError(w, 500, "audit_query", err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
