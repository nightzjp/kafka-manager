package auth

import (
	"testing"
	"time"
)

func TestPasswordHashAndVerify(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "correct horse battery staple" {
		t.Fatal("password was stored in plain text")
	}
	if !VerifyPassword(hash, "correct horse battery staple") {
		t.Fatal("VerifyPassword() rejected the correct password")
	}
	if VerifyPassword(hash, "wrong password") {
		t.Fatal("VerifyPassword() accepted a wrong password")
	}
}

func TestSessionRoundTripAndExpiry(t *testing.T) {
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	manager := NewSessionManager([]byte("a-secret-key-with-at-least-32-bytes"), time.Hour)
	manager.now = func() time.Time { return now }

	token, err := manager.Create("admin")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	username, err := manager.Verify(token)
	if err != nil || username != "admin" {
		t.Fatalf("Verify() = %q, %v; want admin, nil", username, err)
	}

	manager.now = func() time.Time { return now.Add(2 * time.Hour) }
	if _, err := manager.Verify(token); err == nil {
		t.Fatal("Verify() accepted an expired session")
	}
}

func TestSessionRejectsTampering(t *testing.T) {
	manager := NewSessionManager([]byte("a-secret-key-with-at-least-32-bytes"), time.Hour)
	token, err := manager.Create("admin")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := manager.Verify(token + "x"); err == nil {
		t.Fatal("Verify() accepted a modified token")
	}
}
