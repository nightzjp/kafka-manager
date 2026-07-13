package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nightzjp/kafka-manager/internal/cluster"
	"github.com/nightzjp/kafka-manager/internal/config"
)

func TestServerRequiresLoginForClusterData(t *testing.T) {
	cfg := testConfig(t)
	server := NewServer(cfg, nil, cluster.NewManager(cluster.KafkaFactory{}), []byte("a-secret-key-with-at-least-32-bytes"))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/clusters", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", response.Code)
	}
}
func TestServerHealthIsPublic(t *testing.T) {
	server := NewServer(testConfig(t), nil, cluster.NewManager(cluster.KafkaFactory{}), []byte("a-secret-key-with-at-least-32-bytes"))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if response.Code != 200 || !bytes.Contains(response.Body.Bytes(), []byte(`"status":"ok"`)) {
		t.Fatalf("health=%d %s", response.Code, response.Body.String())
	}
}

func TestServerListsConfiguredOfflineCluster(t *testing.T) {
	cfg := testConfig(t)
	server := NewServer(cfg, nil, cluster.NewManager(cluster.KafkaFactory{}), []byte("a-secret-key-with-at-least-32-bytes"))
	login := httptest.NewRecorder()
	server.ServeHTTP(login, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin","password":"secret"}`)))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/clusters", nil)
	request.AddCookie(login.Result().Cookies()[0])
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Items []struct {
			ID     string `json:"id"`
			Online bool   `json:"online"`
		} `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].ID != "dev" || body.Items[0].Online {
		t.Fatalf("body=%+v", body)
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"brokers":0`)) || !bytes.Contains(response.Body.Bytes(), []byte(`"topics":0`)) {
		t.Fatalf("summary counters missing: %s", response.Body.String())
	}
}

func TestConfigAPIHidesKafkaPassword(t *testing.T) {
	cfg := testConfig(t)
	password := cfg.Server.Password
	cfg.Clusters[0].Security.Password = "top-secret"
	server := NewServer(cfg, nil, cluster.NewManager(cluster.KafkaFactory{}), []byte("a-secret-key-with-at-least-32-bytes"))
	login := httptest.NewRecorder()
	server.ServeHTTP(login, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin","password":"secret"}`)))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/config", nil)
	request.AddCookie(login.Result().Cookies()[0])
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != 200 {
		t.Fatalf("status=%d", response.Code)
	}
	if bytes.Contains(response.Body.Bytes(), []byte("top-secret")) || bytes.Contains(response.Body.Bytes(), []byte(password)) {
		t.Fatalf("credential leaked: %s", response.Body.String())
	}
}

func testConfig(t *testing.T) config.Config {
	t.Helper()
	return config.Config{Server: config.ServerConfig{Username: "admin", Password: "secret", SessionHours: 12}, Clusters: []config.ClusterConfig{{ID: "dev", Name: "开发环境", Brokers: []string{"localhost:9092"}}}}
}
