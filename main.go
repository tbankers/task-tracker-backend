package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	api "github.com/tbankers/task-tracker-backend/api"
)

type TaskTrackerServer struct{
	
}

func (s *TaskTrackerServer) GetUserById(w http.ResponseWriter, r *http.Request, id string) {
	
}
func main() {
	serverImpl := &TaskTrackerServer{}

	r := chi.NewRouter()

	api.HandlerFromMux(serverImpl, r)

	fmt.Println("🚀 Server is running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		panic(fmt.Sprintf("Failed to boot the HTTP server: %v", err))
	}
}
