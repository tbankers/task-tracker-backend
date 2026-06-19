package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		duration := time.Since(start)
		fmt.Printf("[API] %s %s → %d (%s)\n", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/auth/") {
			next.ServeHTTP(w, r)
			return
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header required")
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authorization format, use: Bearer <token>")
			return
		}
		claims, err := validateToken(parts[1])
		if err != nil {
			sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token")
			return
		}
		ctx := context.WithValue(r.Context(), contextKeyUserID, claims.UserID)
		ctx = context.WithValue(ctx, contextKeyEmail, claims.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := getEnv("CORS_ORIGIN", "https://localhost")
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getUserID(r *http.Request) string {
	if v := r.Context().Value(contextKeyUserID); v != nil {
		return v.(string)
	}
	return ""
}
