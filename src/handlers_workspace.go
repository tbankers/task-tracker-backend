package main

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tbankers/task-tracker-backend/api"
	db "github.com/tbankers/task-tracker-backend/db"
)

func (s *TaskTrackerServer) CreateWorkspace(w http.ResponseWriter, r *http.Request) {
	var body api.CreateWorkspaceJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	userIDStr := getUserID(r)
	var createdBy uuid.UUID
	if userIDStr != "" {
		createdBy = uuid.MustParse(userIDStr)
	}
	wsID, err := s.Queries.CreateWorkspace(r.Context(), db.CreateWorkspaceParams{
		Title:     pgtype.Text{String: body.Title, Valid: true},
		CreatedBy: &createdBy,
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if userIDStr != "" {
		_, _ = s.Queries.AddMember(r.Context(), db.AddMemberParams{
			UserID:      createdBy,
			WorkspaceID: wsID,
			Role:        db.NullMemberRole{MemberRole: db.MemberRoleAdministrator, Valid: true},
		})
	}
	jsonWrite(w, http.StatusCreated, map[string]interface{}{
		"workspace_id": wsID.String(),
		"title":        body.Title,
		"created_by":   createdBy.String(),
	})
}

func (s *TaskTrackerServer) GetUserWorkspaceById(w http.ResponseWriter, r *http.Request, workspaceId uuid.UUID) {
	ws, err := s.Queries.GetWorkspaceById(r.Context(), workspaceId)
	if err != nil {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "Workspace не найден")
		return
	}
	jsonWrite(w, http.StatusOK, map[string]interface{}{
		"workspace_id": ws.WorkspaceID.String(),
		"title":        ws.Title.String,
		"created_at":   ws.CreatedAt.Time,
		"created_by":   ws.CreatedBy,
	})
}

func (s *TaskTrackerServer) EditWorkspace(w http.ResponseWriter, r *http.Request, workspaceId uuid.UUID) {
	var body api.EditWorkspaceJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	if body.Title != nil {
		err := s.Queries.EditWorkspace(r.Context(), db.EditWorkspaceParams{
			Title:       pgtype.Text{String: *body.Title, Valid: true},
			WorkspaceID: workspaceId,
		})
		if err != nil {
			sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
	}
	jsonWrite(w, http.StatusOK, map[string]string{"message": "Workspace обновлён"})
}

func (s *TaskTrackerServer) DeleteWorkspace(w http.ResponseWriter, r *http.Request, workspaceId uuid.UUID) {
	err := s.Queries.DeleteWorkspace(r.Context(), workspaceId)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
