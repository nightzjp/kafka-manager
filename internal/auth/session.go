package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type SessionManager struct {
	key []byte
	ttl time.Duration
	now func() time.Time
}

type sessionClaims struct {
	Username  string `json:"sub"`
	ExpiresAt int64  `json:"exp"`
}

func NewSessionManager(key []byte, ttl time.Duration) *SessionManager {
	return &SessionManager{key: append([]byte(nil), key...), ttl: ttl, now: time.Now}
}

func (m *SessionManager) Create(username string) (string, error) {
	if len(m.key) < 32 {
		return "", fmt.Errorf("session key must contain at least 32 bytes")
	}
	payload, err := json.Marshal(sessionClaims{Username: username, ExpiresAt: m.now().Add(m.ttl).Unix()})
	if err != nil {
		return "", fmt.Errorf("encode session: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + base64.RawURLEncoding.EncodeToString(m.sign(encoded)), nil
}

func (m *SessionManager) Verify(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid session token")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(signature, m.sign(parts[0])) {
		return "", fmt.Errorf("invalid session signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("decode session: %w", err)
	}
	var claims sessionClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("decode session claims: %w", err)
	}
	if claims.Username == "" || !m.now().Before(time.Unix(claims.ExpiresAt, 0)) {
		return "", fmt.Errorf("session expired")
	}
	return claims.Username, nil
}

func (m *SessionManager) sign(value string) []byte {
	mac := hmac.New(sha256.New, m.key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}
