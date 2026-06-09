package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	api "github.com/tbankers/task-tracker-backend/api"
	db "github.com/tbankers/task-tracker-backend/db"
)

type TaskTrackerServer struct {
	Queries *db.Queries
	DBConn  *pgx.Conn
}

func (s *TaskTrackerServer) GetWorkspaceBoards(w http.ResponseWriter, r *http.Request, workspaceID uuid.UUID) {

	boardIDs, err := s.Queries.GetWorkspaceBoards(r.Context(), &workspaceID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	var response []api.Board
	for _, id := range boardIDs {
		idCopy := id
		response = append(response, api.Board{
			Id:          &idCopy,
			WorkspaceId: &workspaceID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (s *TaskTrackerServer) CreateBoard(w http.ResponseWriter, r *http.Request, workspaceID uuid.UUID) {

	var body api.CreateBoardJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}

	mockUserID := uuid.New()

	boardID, err := s.Queries.CreateBoard(r.Context(), db.CreateBoardParams{
		Name:        pgtype.Text{String: body.Name, Valid: true},
		WorkspaceID: &workspaceID,
		CreatedBy:   &mockUserID,
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	now := time.Now()
	response := api.Board{
		Id:          &boardID,
		Name:        &body.Name,
		WorkspaceId: &workspaceID,
		CreatedAt:   &now,
		CreatedBy:   &mockUserID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (s *TaskTrackerServer) GetBoardTasks(w http.ResponseWriter, r *http.Request, boardID uuid.UUID) {

	taskIDs, err := s.Queries.GetBoardTasks(r.Context(), &boardID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	var response []api.Task
	for _, id := range taskIDs {
		idCopy := id
		response = append(response, api.Task{
			Id:      &idCopy,
			BoardId: &boardID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (s *TaskTrackerServer) CreateTask(w http.ResponseWriter, r *http.Request, boardID uuid.UUID) {

	var body api.CreateTaskJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}

	taskID, err := s.Queries.CreateTask(r.Context(), db.CreateTaskParams{
		Name:        pgtype.Text{String: body.Name, Valid: true},
		Description: pgtype.Text{String: *body.Description, Valid: true},
		AssignedID:  body.AssignedId,
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	now := time.Now()
	defaultStatus := api.TaskStatusToDo

	response := api.Task{
		Id:          &taskID,
		BoardId:     &boardID,
		Name:        &body.Name,
		Description: body.Description,
		Status:      &defaultStatus,
		AssignedId:  body.AssignedId,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (s *TaskTrackerServer) UpdateTask(w http.ResponseWriter, r *http.Request, taskID uuid.UUID) {
	

	var body api.UpdateTaskJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}

	var sqlcStatus db.TaskStatus
	if body.Status != nil {
		sqlcStatus = db.TaskStatus(*body.Status)
	}

	err := s.Queries.UpdateTask(r.Context(), db.UpdateTaskParams{
		ID:          taskID,
		Name:        pgtype.Text{String: *body.Name, Valid: true},
		Description: pgtype.Text{String: *body.Description, Valid: true},
		AssignedID:  body.AssignedId,
		Status:      &sqlcStatus,
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *TaskTrackerServer) DeleteTask(w http.ResponseWriter, r *http.Request, taskID uuid.UUID) {

	err := s.Queries.DeleteTask(r.Context(), taskID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func sendError(w http.ResponseWriter, statusCode int, code string, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(api.ErrorResponse{
		Code:    code,
		Message: msg,
	})
}

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/tasktracker?sslmode=disable"
	}

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		panic(fmt.Sprintf("Не удалось подключиться к БД: %v", err))
	}
	defer conn.Close(ctx)

	queries := db.New(conn)

	serverImpl := &TaskTrackerServer{
		Queries: queries,
		DBConn:  conn,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	api.HandlerFromMux(serverImpl, r)

	port := ":5000"
	fmt.Printf("🚀 Task Tracker api запущен на http://localhost%s\n", port)
	if err := http.ListenAndServe(port, r); err != nil {
		panic(err)
	}
}
