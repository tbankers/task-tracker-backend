package main

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/tbankers/task-tracker-backend/api"
	db "github.com/tbankers/task-tracker-backend/db"
)

func (s *TaskTrackerServer) AddMember(w http.ResponseWriter, r *http.Request, workspaceId uuid.UUID) {
	var body api.AddMemberJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	role := db.NullMemberRole{MemberRole: db.MemberRoleMember, Valid: true}
	if body.Role != nil {
		role = db.NullMemberRole{MemberRole: db.MemberRole(*body.Role), Valid: true}
	}
	_, err := s.Queries.GetWorkspaceById(r.Context(), workspaceId)
	if err != nil {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "Воркспейс не найден")
		return
	}

	_, err = s.Queries.GetUserById(r.Context(), body.UserId)
	if err != nil {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "Пользователь не найден")
		return
	}

	_, err = s.Queries.AddMember(r.Context(), db.AddMemberParams{
		UserID:      body.UserId,
		WorkspaceID: workspaceId,
		Role:        role,
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	jsonWrite(w, http.StatusCreated, map[string]string{"message": "Участник добавлен"})
}

func (s *TaskTrackerServer) ListMembers(w http.ResponseWriter, r *http.Request, workspaceId uuid.UUID) {
	members, err := s.Queries.GetWorkspaceMembers(r.Context(), workspaceId)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	var result []map[string]interface{}
	for _, m := range members {
		result = append(result, map[string]interface{}{
			"user_id":      m.UserID.String(),
			"workspace_id": m.WorkspaceID.String(),
			"role":         string(m.Role.MemberRole),
			"username":     m.Username,
			"email":        m.Email,
		})
	}
	if result == nil {
		result = []map[string]interface{}{}
	}
	jsonWrite(w, http.StatusOK, result)
}

func (s *TaskTrackerServer) KickUser(w http.ResponseWriter, r *http.Request, workspaceId uuid.UUID, userId uuid.UUID) {
	err := s.Queries.KickUser(r.Context(), db.KickUserParams{
		UserID:      userId,
		WorkspaceID: workspaceId,
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *TaskTrackerServer) ManageMember(w http.ResponseWriter, r *http.Request, workspaceId uuid.UUID, userId uuid.UUID) {
	var body api.ManageMemberJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	err := s.Queries.ManageMember(r.Context(), db.ManageMemberParams{
		Role:   db.NullMemberRole{MemberRole: db.MemberRole(body.Role), Valid: true},
		UserID: userId,
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	jsonWrite(w, http.StatusOK, map[string]string{"message": "Роль участника обновлена"})
}

func (s *TaskTrackerServer) GetMemberRoleById(w http.ResponseWriter, r *http.Request, workspaceId uuid.UUID, userId uuid.UUID) {
	role, err := s.Queries.GetMemberRoleById(r.Context(), userId)
	if err != nil {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "Участник не найден")
		return
	}
	jsonWrite(w, http.StatusOK, map[string]interface{}{
		"role": string(role.MemberRole),
	})
}
