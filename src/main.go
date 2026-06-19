package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"

	"github.com/tbankers/task-tracker-backend/api"
	db "github.com/tbankers/task-tracker-backend/db"
)

type TaskTrackerServer struct {
	Queries *db.Queries
	DBConn  *pgx.Conn
}

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5433/tasktracker?sslmode=disable"
	}

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		panic(fmt.Sprintf("Не удалось подключиться к БД: %v", err))
	}
	defer func() { _ = conn.Close(ctx) }()

	queries := db.New(conn)

	serverImpl := &TaskTrackerServer{
		Queries: queries,
		DBConn:  conn,
	}

	r := chi.NewRouter()
	r.Use(requestLogger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		api.HandlerFromMux(serverImpl, r)
	})

	port := getEnv("APP_PORT", "8080")
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	fmt.Printf("Task Tracker api запущен на http://localhost%s\n", port)
	if err := http.ListenAndServe(port, r); err != nil {
		panic(err)
	}
}
