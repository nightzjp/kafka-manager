package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nightzjp/kafka-manager/internal/auth"
)

func TestLoginAndCurrentUser(t *testing.T) {
	hash, err := auth.HashPassword("secret-password")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewAuthHandler("admin", hash, auth.NewSessionManager([]byte("a-secret-key-with-at-least-32-bytes"), time.Hour))

	login := httptest.NewRecorder()
	handler.ServeHTTP(login, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin","password":"secret-password"}`)))
	if login.Code != http.StatusNoContent {
		t.Fatalf("login status = %d, want 204; body=%s", login.Code, login.Body.String())
	}
	cookies := login.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("unexpected session cookie: %+v", cookies)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	request.AddCookie(cookies[0])
	me := httptest.NewRecorder()
	handler.ServeHTTP(me, request)
	if me.Code != http.StatusOK || !bytes.Contains(me.Body.Bytes(), []byte(`"username":"admin"`)) {
		t.Fatalf("me response = %d %s", me.Code, me.Body.String())
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	hash, err := auth.HashPassword("secret-password")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewAuthHandler("admin", hash, auth.NewSessionManager([]byte("a-secret-key-with-at-least-32-bytes"), time.Hour))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin","password":"wrong"}`)))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
}

func TestCurrentUserRequiresSession(t *testing.T) {
	handler := NewAuthHandler("admin", "hash", auth.NewSessionManager([]byte("a-secret-key-with-at-least-32-bytes"), time.Hour))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
}

func TestLoginRateLimitsRepeatedFailures(t *testing.T) {
	hash, err := auth.HashPassword("secret-password")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewAuthHandler("admin", hash, auth.NewSessionManager([]byte("a-secret-key-with-at-least-32-bytes"), time.Hour))
	for attempt := 1; attempt <= 6; attempt++ {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin","password":"wrong"}`))
		request.RemoteAddr = "192.0.2.10:1234"
		handler.ServeHTTP(response, request)
		if attempt <= 5 && response.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d, want 401", attempt, response.Code)
		}
		if attempt == 6 && response.Code != http.StatusTooManyRequests {
			t.Fatalf("attempt %d status = %d, want 429", attempt, response.Code)
		}
	}
}
