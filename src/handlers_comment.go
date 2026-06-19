package main

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tbankers/task-tracker-backend/db"
)

func (s *TaskTrackerServer) GetBoardComments(w http.ResponseWriter, r *http.Request, boardId uuid.UUID) {
	wsID, err := s.Queries.GetBoardWorkspaceID(r.Context(), boardId)
	if err != nil {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "Доска не найдена")
		return
	}
	if err := checkAccess(r.Context(), s.Queries, r, wsID); err != nil {
		sendError(w, http.StatusForbidden, "FORBIDDEN", "Нет доступа к воркспейсу")
		return
	}
	comments, err := s.Queries.GetCommentsByBoard(r.Context(), &boardId)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	var response []map[string]interface{}
	for _, c := range comments {
		item := map[string]interface{}{
			"comment_id": c.CommentID,
			"board_id":   c.BoardID,
			"content":    c.Content,
			"sent_at":    c.SentAt.Time,
		}
		if c.AuthorID != nil {
			item["author_id"] = c.AuthorID
		}
		response = append(response, item)
	}
	jsonWrite(w, http.StatusOK, response)
}

func (s *TaskTrackerServer) CreateBoardComment(w http.ResponseWriter, r *http.Request, boardId uuid.UUID) {
	wsID, err := s.Queries.GetBoardWorkspaceID(r.Context(), boardId)
	if err != nil {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "Доска не найдена")
		return
	}
	if err := checkAccess(r.Context(), s.Queries, r, wsID); err != nil {
		sendError(w, http.StatusForbidden, "FORBIDDEN", "Нет доступа к воркспейсу")
		return
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	userIDStr := getUserID(r)
	var authorID *uuid.UUID
	if userIDStr != "" {
		u := uuid.MustParse(userIDStr)
		authorID = &u
	}
	comment, err := s.Queries.CreateComment(r.Context(), db.CreateCommentParams{
		BoardID:  &boardId,
		AuthorID: authorID,
		Content:  pgtype.Text{String: body.Content, Valid: true},
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	result := map[string]interface{}{
		"comment_id": comment.CommentID,
		"board_id":   comment.BoardID,
		"content":    comment.Content,
		"sent_at":    comment.SentAt.Time,
	}
	if comment.AuthorID != nil {
		result["author_id"] = comment.AuthorID
	}
	jsonWrite(w, http.StatusCreated, result)
}

func (s *TaskTrackerServer) UpdateBoardComment(w http.ResponseWriter, r *http.Request, boardId uuid.UUID, commentId int) {
	wsID, err := s.Queries.GetBoardWorkspaceID(r.Context(), boardId)
	if err != nil {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "Доска не найдена")
		return
	}
	if err := checkAccess(r.Context(), s.Queries, r, wsID); err != nil {
		sendError(w, http.StatusForbidden, "FORBIDDEN", "Нет доступа к воркспейсу")
		return
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	err = s.Queries.UpdateComment(r.Context(), db.UpdateCommentParams{
		Content:   pgtype.Text{String: body.Content, Valid: true},
		CommentID: int32(commentId),
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	comment, err := s.Queries.GetCommentById(r.Context(), int32(commentId))
	if err != nil {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "Комментарий не найден")
		return
	}
	result := map[string]interface{}{
		"comment_id": comment.CommentID,
		"board_id":   comment.BoardID,
		"content":    comment.Content,
		"sent_at":    comment.SentAt.Time,
	}
	if comment.AuthorID != nil {
		result["author_id"] = comment.AuthorID
	}
	jsonWrite(w, http.StatusOK, result)
}

func (s *TaskTrackerServer) DeleteBoardComment(w http.ResponseWriter, r *http.Request, boardId uuid.UUID, commentId int) {
	wsID, err := s.Queries.GetBoardWorkspaceID(r.Context(), boardId)
	if err != nil {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "Доска не найдена")
		return
	}
	if err := checkAccess(r.Context(), s.Queries, r, wsID); err != nil {
		sendError(w, http.StatusForbidden, "FORBIDDEN", "Нет доступа к воркспейсу")
		return
	}
	err = s.Queries.DeleteComment(r.Context(), int32(commentId))
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
