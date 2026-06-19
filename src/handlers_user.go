package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/tbankers/task-tracker-backend/api"
	db "github.com/tbankers/task-tracker-backend/db"
)

func (s *TaskTrackerServer) GetUserByEmail(w http.ResponseWriter, r *http.Request, params api.GetUserByEmailParams) {
	user, err := s.Queries.GetUserByEmail(r.Context(), string(params.Email))
	if err != nil {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "Пользователь не найден")
		return
	}
	now := user.CreatedAt.Time
	jsonWrite(w, http.StatusOK, map[string]interface{}{
		"user_id":    user.UserID.String(),
		"email":      user.Email,
		"username":   user.Username,
		"created_at": now,
	})
}

func (s *TaskTrackerServer) CreateUser(w http.ResponseWriter, r *http.Request) {
	var body api.CreateUserJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "HASH_ERROR", "Ошибка хеширования пароля")
		return
	}
	userID, err := s.Queries.CreateUser(r.Context(), db.CreateUserParams{
		Email:        string(body.Email),
		Username:     body.Username,
		PasswordHash: string(hash),
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	jsonWrite(w, http.StatusCreated, map[string]interface{}{
		"user_id":  userID.String(),
		"email":    string(body.Email),
		"username": body.Username,
	})
}

func (s *TaskTrackerServer) GetUserById(w http.ResponseWriter, r *http.Request, userId uuid.UUID) {
	user, err := s.Queries.GetUserById(r.Context(), userId)
	if err != nil {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "Пользователь не найден")
		return
	}
	now := user.CreatedAt.Time
	jsonWrite(w, http.StatusOK, map[string]interface{}{
		"user_id":    user.UserID.String(),
		"email":      user.Email,
		"username":   user.Username,
		"created_at": now,
	})
}

func (s *TaskTrackerServer) ChangePassword(w http.ResponseWriter, r *http.Request, userId uuid.UUID) {
	var body api.ChangePasswordJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "HASH_ERROR", "Ошибка хеширования пароля")
		return
	}
	err = s.Queries.ChangePassword(r.Context(), db.ChangePasswordParams{
		PasswordHash: string(hash),
		UserID:       userId,
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	user, err := s.Queries.GetUserById(r.Context(), userId)
	if err == nil {
		emailBody := passwordChangedEmailHTML(user.Email)
		if err := sendEmail(user.Email, "Пароль изменён - Task Tracker", emailBody); err != nil {
			fmt.Printf("[EMAIL ERROR] %v\n", err)
		}
	}
	jsonWrite(w, http.StatusOK, map[string]string{"message": "Пароль изменён"})
}

func (s *TaskTrackerServer) GetUserWorkspace(w http.ResponseWriter, r *http.Request, userId uuid.UUID) {
	workspaces, err := s.Queries.GetAllUserWorkspaces(r.Context(), &userId)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	var result []map[string]interface{}
	for _, ws := range workspaces {
		result = append(result, map[string]interface{}{
			"workspace_id": ws.WorkspaceID.String(),
			"title":        ws.Title.String,
			"created_by":   ws.CreatedBy,
			"created_at":   ws.CreatedAt.Time,
		})
	}
	if result == nil {
		result = []map[string]interface{}{}
	}
	jsonWrite(w, http.StatusOK, result)
}
