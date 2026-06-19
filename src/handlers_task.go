package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tbankers/task-tracker-backend/db"
)

func (s *TaskTrackerServer) GetTasksFromBoard(w http.ResponseWriter, r *http.Request, boardId uuid.UUID) {
	tasks, err := s.Queries.GetTasksFromBoard(r.Context(), boardId)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	var response []map[string]interface{}
	for _, t := range tasks {
		task := map[string]interface{}{
			"id":       t.TaskID,
			"board_id": boardId,
		}
		if t.ColumnID != nil {
			task["column_id"] = t.ColumnID
		}
		if t.Title.Valid {
			task["title"] = t.Title.String
		}
		if t.Description.Valid {
			task["description"] = t.Description.String
		}
		if t.AssignedID != nil {
			task["assigned_id"] = t.AssignedID
		}
		if t.CreatedBy != nil {
			task["created_by"] = t.CreatedBy
		}
		createdAt := t.CreatedAt.Time
		updatedAt := t.UpdatedAt.Time
		task["created_at"] = createdAt
		task["updated_at"] = updatedAt

		blockpoints, err := s.Queries.GetTaskBlockpoints(r.Context(), t.TaskID)
		if err == nil {
			task["blocked_by"] = blockpoints
		} else {
			task["blocked_by"] = []int32{}
		}
		if t.StartDate.Valid {
			task["start_date"] = t.StartDate.Time.Format("2006-01-02")
		}
		if t.DueDate.Valid {
			task["due_date"] = t.DueDate.Time.Format("2006-01-02")
		}

		response = append(response, task)
	}
	jsonWrite(w, http.StatusOK, response)
}

func (s *TaskTrackerServer) CreateTask(w http.ResponseWriter, r *http.Request, boardId uuid.UUID) {
	var body struct {
		Title       string     `json:"title"`
		Description string     `json:"description"`
		ColumnID    *uuid.UUID `json:"column_id"`
		AssignedID  *uuid.UUID `json:"assigned_id"`
		StartDate   *string    `json:"start_date"`
		DueDate     *string    `json:"due_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	userIDStr := getUserID(r)
	var createdBy *uuid.UUID
	if userIDStr != "" {
		u := uuid.MustParse(userIDStr)
		createdBy = &u
	}
	var startDate pgtype.Date
	if body.StartDate != nil && *body.StartDate != "" {
		t, err := time.Parse("2006-01-02", *body.StartDate)
		if err == nil {
			startDate = pgtype.Date{Time: t, Valid: true}
		}
	}
	var dueDate pgtype.Date
	if body.DueDate != nil && *body.DueDate != "" {
		t, err := time.Parse("2006-01-02", *body.DueDate)
		if err == nil {
			dueDate = pgtype.Date{Time: t, Valid: true}
		}
	}
	taskID, err := s.Queries.CreateTask(r.Context(), db.CreateTaskParams{
		ColumnID:    body.ColumnID,
		CreatedBy:   createdBy,
		Title:       pgtype.Text{String: body.Title, Valid: true},
		Description: pgtype.Text{String: body.Description, Valid: body.Description != ""},
		AssignedID:  body.AssignedID,
		StartDate:   startDate,
		DueDate:     dueDate,
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	now := time.Now()
	resp := map[string]interface{}{
		"id":         taskID,
		"board_id":   boardId.String(),
		"title":      body.Title,
		"created_at": now,
		"updated_at": now,
		"blocked_by": []int32{},
	}
	if body.ColumnID != nil {
		resp["column_id"] = body.ColumnID
	}
	if startDate.Valid {
		resp["start_date"] = startDate.Time.Format("2006-01-02")
	}
	if dueDate.Valid {
		resp["due_date"] = dueDate.Time.Format("2006-01-02")
	}
	jsonWrite(w, http.StatusCreated, resp)
}

func (s *TaskTrackerServer) ChangeTaskStatus(w http.ResponseWriter, r *http.Request, taskId int) {
	var body struct {
		Title       *string    `json:"title"`
		Description *string    `json:"description"`
		AssignedID  *uuid.UUID `json:"assigned_id"`
		ColumnID    *uuid.UUID `json:"column_id"`
		StartDate   *string    `json:"start_date"`
		DueDate     *string    `json:"due_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	params := db.UpdateTaskParams{
		TaskID: int32(taskId),
	}
	if body.Title != nil {
		params.Title = pgtype.Text{String: *body.Title, Valid: true}
	}
	if body.Description != nil {
		params.Description = pgtype.Text{String: *body.Description, Valid: true}
	}
	params.AssignedID = body.AssignedID
	params.ColumnID = body.ColumnID
	if body.StartDate != nil && *body.StartDate != "" {
		t, err := time.Parse("2006-01-02", *body.StartDate)
		if err == nil {
			params.StartDate = pgtype.Date{Time: t, Valid: true}
		}
	}
	if body.DueDate != nil && *body.DueDate != "" {
		t, err := time.Parse("2006-01-02", *body.DueDate)
		if err == nil {
			params.DueDate = pgtype.Date{Time: t, Valid: true}
		}
	}
	err := s.Queries.UpdateTask(r.Context(), params)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	jsonWrite(w, http.StatusOK, map[string]string{"message": "Задача обновлена"})
}

func (s *TaskTrackerServer) DeleteTask(w http.ResponseWriter, r *http.Request, taskId int) {
	_ = s.Queries.DeleteAllBlockpointsForTask(r.Context(), int32(taskId))
	err := s.Queries.DeleteTask(r.Context(), int32(taskId))
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *TaskTrackerServer) GetTaskBlockpoints(w http.ResponseWriter, r *http.Request, taskId int) {
	blockpoints, err := s.Queries.GetTaskBlockpoints(r.Context(), int32(taskId))
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	jsonWrite(w, http.StatusOK, map[string]interface{}{
		"task_id":    taskId,
		"blocked_by": blockpoints,
	})
}

func (s *TaskTrackerServer) AddBlockpoint(w http.ResponseWriter, r *http.Request, taskId int) {
	var body struct {
		BlockedByTaskId int `json:"blocked_by_task_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	if body.BlockedByTaskId == taskId {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Задача не может блокировать сама себя")
		return
	}
	err := s.Queries.AddBlockpoint(r.Context(), db.AddBlockpointParams{
		TaskID:          int32(taskId),
		BlockedByTaskID: int32(body.BlockedByTaskId),
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	jsonWrite(w, http.StatusCreated, map[string]string{"message": "Блокпоинт добавлен"})
}

func (s *TaskTrackerServer) RemoveBlockpoint(w http.ResponseWriter, r *http.Request, taskId int, blockedByTaskId int) {
	err := s.Queries.RemoveBlockpoint(r.Context(), db.RemoveBlockpointParams{
		TaskID:          int32(taskId),
		BlockedByTaskID: int32(blockedByTaskId),
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
