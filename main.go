package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	// Correct import path for your generated code
	api "github.com/tbankers/task-tracker-backend/API"
)

// TaskTrackerServer implements the generated api.ServerInterface
type TaskTrackerServer struct{}

// GetUsersId handles GET /users/{id}
func (s *TaskTrackerServer) GetUserById(w http.ResponseWriter, r *http.Request, id string) {
	// Parse the path string into a Google UUID object
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "Invalid user UUID format", http.StatusBadRequest)
		return
	}

	name := "Иван Иванов"
	user := api.User{
		Id:   &parsedUUID, // Assigned as a pointer to the UUID
		Name: &name,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

// GetTasksId handles GET /tasks/{id}
func (s *TaskTrackerServer) GetTaskById(w http.ResponseWriter, r *http.Request, id string) {
	// Parse the path string into a Google UUID object
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "Invalid task UUID format", http.StatusBadRequest)
		return
	}

	title := "Изучить oapi-codegen"
	content := "Разобраться с импортами и запустить тестовый main.go"

	task := api.Task{
		Id:      &parsedUUID,
		Title:   &title,
		Content: &content,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(task)
}

func main() {
	serverImpl := &TaskTrackerServer{}

	r := chi.NewRouter()
	// r.Use(middleware.Logger)

	// This call will compile successfully now because your struct satisfies the interface rules
	api.HandlerFromMux(serverImpl, r)

	fmt.Println("🚀 Server is running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		panic(fmt.Sprintf("Failed to boot the HTTP server: %v", err))
	}
}
