package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/tbankers/task-tracker-backend/api"
	db "github.com/tbankers/task-tracker-backend/db"
)

func (s *TaskTrackerServer) GetWorkspaceBoards(w http.ResponseWriter, r *http.Request, workspaceId uuid.UUID) {
	if err := checkAccess(r.Context(), s.Queries, r, workspaceId); err != nil {
		sendError(w, http.StatusForbidden, "FORBIDDEN", "Нет доступа к воркспейсу")
		return
	}
	boards, err := s.Queries.GetWorkspaceBoards(r.Context(), &workspaceId)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	var response []map[string]interface{}
	for _, b := range boards {
		board := map[string]interface{}{
			"id":           b.BoardID.String(),
			"workspace_id": workspaceId.String(),
		}
		if b.Title.Valid {
			board["title"] = b.Title.String
		}
		if b.CreatedAt.Valid {
			board["created_at"] = b.CreatedAt.Time
		}
		if b.CreatedBy != nil {
			board["created_by"] = b.CreatedBy.String()
		}
		response = append(response, board)
	}
	jsonWrite(w, http.StatusOK, response)
}

func (s *TaskTrackerServer) CreateBoard(w http.ResponseWriter, r *http.Request, workspaceId uuid.UUID) {
	if err := checkAccess(r.Context(), s.Queries, r, workspaceId); err != nil {
		sendError(w, http.StatusForbidden, "FORBIDDEN", "Нет доступа к воркспейсу")
		return
	}
	var body api.CreateBoardJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	userIDStr := getUserID(r)
	var createdBy uuid.UUID
	if userIDStr != "" {
		createdBy = uuid.MustParse(userIDStr)
	}
	boardID, err := s.Queries.CreateBoard(r.Context(), db.CreateBoardParams{
		Title:       pgtype.Text{String: body.Title, Valid: true},
		WorkspaceID: &workspaceId,
		CreatedBy:   &createdBy,
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	now := time.Now()
	jsonWrite(w, http.StatusCreated, map[string]interface{}{
		"id":           boardID.String(),
		"title":        body.Title,
		"workspace_id": workspaceId.String(),
		"created_at":   now,
		"created_by":   createdBy.String(),
	})
}

func (s *TaskTrackerServer) EditBoard(w http.ResponseWriter, r *http.Request, boardId uuid.UUID) {
	wsID, err := s.Queries.GetBoardWorkspaceID(r.Context(), boardId)
	if err != nil {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "Доска не найдена")
		return
	}
	if err := checkAccess(r.Context(), s.Queries, r, wsID); err != nil {
		sendError(w, http.StatusForbidden, "FORBIDDEN", "Нет доступа к воркспейсу")
		return
	}
	var body api.EditBoardJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	if body.Title != nil {
		err := s.Queries.EditBoard(r.Context(), db.EditBoardParams{
			Title:   pgtype.Text{String: *body.Title, Valid: true},
			BoardID: boardId,
		})
		if err != nil {
			sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
	}
	jsonWrite(w, http.StatusOK, map[string]string{"message": "Board обновлён"})
}

func (s *TaskTrackerServer) DeleteBoard(w http.ResponseWriter, r *http.Request, boardId uuid.UUID) {
	wsID, err := s.Queries.GetBoardWorkspaceID(r.Context(), boardId)
	if err != nil {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "Доска не найдена")
		return
	}
	if err := checkAccess(r.Context(), s.Queries, r, wsID); err != nil {
		sendError(w, http.StatusForbidden, "FORBIDDEN", "Нет доступа к воркспейсу")
		return
	}
	err = s.Queries.DeleteBoard(r.Context(), boardId)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *TaskTrackerServer) GetBoardColumns(w http.ResponseWriter, r *http.Request, boardId uuid.UUID) {
	wsID, err := s.Queries.GetBoardWorkspaceID(r.Context(), boardId)
	if err != nil {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "Доска не найдена")
		return
	}
	if err := checkAccess(r.Context(), s.Queries, r, wsID); err != nil {
		sendError(w, http.StatusForbidden, "FORBIDDEN", "Нет доступа к воркспейсу")
		return
	}
	columns, err := s.Queries.GetColumnsFromBoard(r.Context(), boardId)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	var response []map[string]interface{}
	for _, c := range columns {
		col := map[string]interface{}{
			"column_id": c.ColumnID.String(),
			"board_id":  c.BoardID.String(),
			"name":      c.Name,
			"position":  c.Position,
		}
		if c.CreatedAt.Valid {
			col["created_at"] = c.CreatedAt.Time
		}
		response = append(response, col)
	}
	jsonWrite(w, http.StatusOK, response)
}

func (s *TaskTrackerServer) CreateColumn(w http.ResponseWriter, r *http.Request, boardId uuid.UUID) {
	wsID, err := s.Queries.GetBoardWorkspaceID(r.Context(), boardId)
	if err != nil {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "Доска не найдена")
		return
	}
	if err := checkAccess(r.Context(), s.Queries, r, wsID); err != nil {
		sendError(w, http.StatusForbidden, "FORBIDDEN", "Нет доступа к воркспейсу")
		return
	}
	var body api.CreateColumnJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	position := int32(0)
	if body.Position != nil {
		position = int32(*body.Position)
	}
	col, err := s.Queries.CreateColumn(r.Context(), db.CreateColumnParams{
		BoardID:  boardId,
		Name:     body.Name,
		Position: position,
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	result := map[string]interface{}{
		"column_id": col.ColumnID.String(),
		"board_id":  col.BoardID.String(),
		"name":      col.Name,
		"position":  col.Position,
	}
	if col.CreatedAt.Valid {
		result["created_at"] = col.CreatedAt.Time
	}
	jsonWrite(w, http.StatusCreated, result)
}

func (s *TaskTrackerServer) DeleteColumn(w http.ResponseWriter, r *http.Request, columnId openapi_types.UUID) {
	wsID, err := s.Queries.GetColumnWorkspaceID(r.Context(), columnId)
	if err != nil {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "Колонка не найдена")
		return
	}
	if err := checkAccess(r.Context(), s.Queries, r, wsID); err != nil {
		sendError(w, http.StatusForbidden, "FORBIDDEN", "Нет доступа к воркспейсу")
		return
	}
	err = s.Queries.DeleteColumn(r.Context(), columnId)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
