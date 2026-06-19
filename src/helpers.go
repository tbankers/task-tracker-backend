package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/tbankers/task-tracker-backend/db"
)

func sendError(w http.ResponseWriter, statusCode int, code string, msg string) {
	fmt.Printf("[ERROR] %d %s: %s\n", statusCode, code, msg)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    code,
		"message": msg,
	})
}

func jsonWrite(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func requireWorkspaceMember(ctx context.Context, queries *db.Queries, userID string, workspaceID uuid.UUID) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id")
	}
	member, err := queries.IsWorkspaceMember(ctx, uid, workspaceID)
	if err != nil || !member {
		return fmt.Errorf("not a member")
	}
	return nil
}

func checkAccess(ctx context.Context, queries *db.Queries, r *http.Request, workspaceID uuid.UUID) error {
	userID := getUserID(r)
	return requireWorkspaceMember(ctx, queries, userID, workspaceID)
}
