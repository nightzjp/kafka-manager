package api

import (
	"crypto/subtle"
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/nightzjp/kafka-manager/internal/auth"
)

const sessionCookieName = "kafka_manager_session"

type AuthHandler struct {
	username string
	password string
	hash     string
	sessions *auth.SessionManager
	mux      *http.ServeMux
	mu       sync.Mutex
	failures map[string]loginFailures
}

type loginFailures struct {
	count   int
	firstAt time.Time
}

func NewAuthHandler(username, password, hash string, sessions *auth.SessionManager) *AuthHandler {
	h := &AuthHandler{username: username, password: password, hash: hash, sessions: sessions, mux: http.NewServeMux(), failures: make(map[string]loginFailures)}
	h.mux.HandleFunc("POST /api/v1/auth/login", h.login)
	h.mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
	h.mux.HandleFunc("GET /api/v1/auth/me", h.me)
	return h
}

func (h *AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	client := clientHost(r.RemoteAddr)
	if h.rateLimited(client, time.Now()) {
		writeError(w, http.StatusTooManyRequests, "too_many_attempts", "登录失败次数过多，请稍后重试")
		return
	}
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&credentials); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "登录请求格式不正确")
		return
	}
	passwordOK := subtle.ConstantTimeCompare([]byte(h.password), []byte(credentials.Password)) == 1
	if h.password == "" {
		passwordOK = auth.VerifyPassword(h.hash, credentials.Password)
	}
	if credentials.Username != h.username || !passwordOK {
		h.recordFailure(client, time.Now())
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "用户名或密码错误")
		return
	}
	h.clearFailures(client)
	token, err := h.sessions.Create(h.username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session_error", "无法创建登录会话")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func clientHost(remoteAddress string) string {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil {
		return host
	}
	return remoteAddress
}

func (h *AuthHandler) rateLimited(client string, now time.Time) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	failure, ok := h.failures[client]
	if !ok {
		return false
	}
	if now.Sub(failure.firstAt) >= 5*time.Minute {
		delete(h.failures, client)
		return false
	}
	return failure.count >= 5
}

func (h *AuthHandler) recordFailure(client string, now time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	failure, ok := h.failures[client]
	if !ok || now.Sub(failure.firstAt) >= 5*time.Minute {
		h.failures[client] = loginFailures{count: 1, firstAt: now}
		return
	}
	failure.count++
	h.failures[client] = failure
}

func (h *AuthHandler) clearFailures(client string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.failures, client)
}

func (h *AuthHandler) logout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1, Expires: time.Unix(1, 0)})
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "请先登录")
		return
	}
	username, err := h.sessions.Verify(cookie.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "登录会话已失效")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"username": username})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}})
}
