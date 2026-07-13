package auth

import (
	"context"
	"net/http"
)

type contextKey string

const usernameContextKey contextKey = "username"

func Middleware(manager *SessionManager, cookieName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		username, err := manager.Verify(cookie.Value)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), usernameContextKey, username)))
	})
}

func Username(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(usernameContextKey).(string)
	return username, ok
}
