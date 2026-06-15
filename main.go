package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/tbankers/task-tracker-backend/api"
	db "github.com/tbankers/task-tracker-backend/db"
)

var jwtSecret []byte

type smtpConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

var smtpCfg smtpConfig

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}
	jwtSecret = []byte(secret)

	port := 587
	fmt.Sscanf(os.Getenv("SMTP_PORT"), "%d", &port)
	smtpCfg = smtpConfig{
		Host:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		Port:     port,
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     getEnv("SMTP_FROM", os.Getenv("SMTP_USERNAME")),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func sendEmail(to, subject, body string) error {
	if smtpCfg.Username == "" || smtpCfg.Password == "" {
		fmt.Printf("[EMAIL STUB] To: %s | Subject: %s\n", to, subject)
		return nil
	}

	headers := make(map[string]string)
	headers["From"] = smtpCfg.From
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"UTF-8\""

	msg := ""
	for k, v := range headers {
		msg += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	msg += "\r\n" + body

	addr := fmt.Sprintf("%s:%d", smtpCfg.Host, smtpCfg.Port)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("net dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, smtpCfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if err = client.StartTLS(&tls.Config{ServerName: smtpCfg.Host}); err != nil {
		return fmt.Errorf("starttls: %w", err)
	}

	auth := smtp.PlainAuth("", smtpCfg.Username, smtpCfg.Password, smtpCfg.Host)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err = client.Mail(smtpCfg.From); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err = w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}

	return client.Quit()
}

type TaskTrackerServer struct {
	Queries *db.Queries
	DBConn  *pgx.Conn
}

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func generateToken(userID, email string) (string, error) {
	claims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func validateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		duration := time.Since(start)
		fmt.Printf("[API] %s %s → %d (%s)\n", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/auth/") || strings.HasPrefix(r.URL.Path, "/frontend") {
			next.ServeHTTP(w, r)
			return
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header required")
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authorization format, use: Bearer <token>")
			return
		}
		claims, err := validateToken(parts[1])
		if err != nil {
			sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token")
			return
		}
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "email", claims.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getUserID(r *http.Request) string {
	if v := r.Context().Value("user_id"); v != nil {
		return v.(string)
	}
	return ""
}

func generateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func sendError(w http.ResponseWriter, statusCode int, code string, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"code":    code,
		"message": msg,
	})
}

func jsonWrite(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// --- Auth handlers ---

func (s *TaskTrackerServer) Register(w http.ResponseWriter, r *http.Request) {
	var body api.RegisterJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	_, err := s.Queries.GetUserByEmail(r.Context(), string(body.Email))
	if err == nil {
		sendError(w, http.StatusConflict, "CONFLICT", "Пользователь с таким email уже существует")
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
	token, err := generateToken(userID.String(), string(body.Email))
	if err != nil {
		sendError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Ошибка генерации токена")
		return
	}
	now := time.Now()
	jsonWrite(w, http.StatusCreated, map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"user_id":    userID.String(),
			"email":      string(body.Email),
			"username":   body.Username,
			"created_at": now,
		},
	})
}

func (s *TaskTrackerServer) Login(w http.ResponseWriter, r *http.Request) {
	var body api.LoginJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	user, err := s.Queries.GetUserByEmail(r.Context(), string(body.Email))
	if err != nil {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Неверный email или пароль")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Неверный email или пароль")
		return
	}
	token, err := generateToken(user.UserID.String(), user.Email)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Ошибка генерации токена")
		return
	}
	now := time.Now()
	jsonWrite(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"user_id":    user.UserID.String(),
			"email":      user.Email,
			"username":   user.Username,
			"created_at": now,
		},
	})
}

func (s *TaskTrackerServer) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var body api.ForgotPasswordJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	user, err := s.Queries.GetUserByEmail(r.Context(), string(body.Email))
	if err != nil {
		jsonWrite(w, http.StatusOK, map[string]string{
			"message": "Если пользователь с таким email существует, письмо с инструкцией отправлено",
		})
		return
	}
	resetToken, err := generateResetToken()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Ошибка генерации токена")
		return
	}
	expiresAt := pgtype.Timestamp{Time: time.Now().Add(1 * time.Hour), Valid: true}
	_, err = s.Queries.CreatePasswordResetToken(r.Context(), db.CreatePasswordResetTokenParams{
		UserID:    user.UserID,
		Token:     resetToken,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", getEnv("FRONTEND_URL", "http://localhost:3000"), resetToken)
	emailBody := fmt.Sprintf(`
		<h2>Сброс пароля</h2>
		<p>Вы запросили сброс пароля для аккаунта %s.</p>
		<p>Перейдите по ссылке для создания нового пароля:</p>
		<p><a href="%s">%s</a></p>
		<p>Ссылка действительна в течение 1 часа.</p>
		<p>Если вы не запрашивали сброс пароля, проигнорируйте это письмо.</p>
	`, user.Email, resetLink, resetLink)

	if err := sendEmail(user.Email, "Сброс пароля - Task Tracker", emailBody); err != nil {
		fmt.Printf("[EMAIL ERROR] %v\n", err)
		sendError(w, http.StatusInternalServerError, "EMAIL_ERROR", err.Error())
		return
	}
	jsonWrite(w, http.StatusOK, map[string]string{
		"message": "Письмо с инструкцией по сбросу пароля отправлено",
	})
}

func (s *TaskTrackerServer) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var body api.ResetPasswordJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	resetToken, err := s.Queries.GetPasswordResetToken(r.Context(), body.Token)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Неверный или просроченный токен")
		return
	}
	if time.Now().After(resetToken.ExpiresAt.Time) {
		s.Queries.DeletePasswordResetToken(r.Context(), body.Token)
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Токен просрочен")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "HASH_ERROR", "Ошибка хеширования пароля")
		return
	}
	err = s.Queries.ChangePassword(r.Context(), db.ChangePasswordParams{
		PasswordHash: string(hash),
		UserID:       resetToken.UserID,
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	s.Queries.DeletePasswordResetToken(r.Context(), body.Token)
	jsonWrite(w, http.StatusOK, map[string]string{
		"message": "Пароль успешно изменён",
	})
}

// --- User handlers ---

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
	jsonWrite(w, http.StatusOK, map[string]string{"message": "Пароль изменён"})
}

func (s *TaskTrackerServer) GetUserWorkspace(w http.ResponseWriter, r *http.Request, userId uuid.UUID) {
	workspaces, err := s.Queries.GetUsersWorkspace(r.Context(), &userId)
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
	jsonWrite(w, http.StatusOK, result)
}

// --- Workspace handlers ---

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

// --- Board handlers ---

func (s *TaskTrackerServer) GetWorkspaceBoards(w http.ResponseWriter, r *http.Request, workspaceId uuid.UUID) {
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
	err := s.Queries.DeleteBoard(r.Context(), boardId)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Task handlers ---

func (s *TaskTrackerServer) GetTasksFromBoard(w http.ResponseWriter, r *http.Request, boardId uuid.UUID) {
	tasks, err := s.Queries.GetTasksFromBoard(r.Context(), &boardId)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	var response []map[string]interface{}
	for _, t := range tasks {
		task := map[string]interface{}{
			"id":       t.TaskID,
			"board_id": t.BoardID,
		}
		if t.Title.Valid {
			task["title"] = t.Title.String
		}
		if t.Description.Valid {
			task["description"] = t.Description.String
		}
		if t.Status.Valid {
			task["status"] = string(t.Status.TaskStatus)
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
		response = append(response, task)
	}
	jsonWrite(w, http.StatusOK, response)
}

func (s *TaskTrackerServer) CreateTask(w http.ResponseWriter, r *http.Request, boardId uuid.UUID) {
	var body api.CreateTaskJSONRequestBody
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
	taskID, err := s.Queries.CreateTask(r.Context(), db.CreateTaskParams{
		BoardID:     &boardId,
		CreatedBy:   createdBy,
		Title:       pgtype.Text{String: body.Title, Valid: true},
		Description: pgtype.Text{},
		AssignedID:  nil,
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	now := time.Now()
	jsonWrite(w, http.StatusCreated, map[string]interface{}{
		"id":         taskID,
		"board_id":   boardId.String(),
		"title":      body.Title,
		"status":     "to_do",
		"created_at": now,
		"updated_at": now,
	})
}

func (s *TaskTrackerServer) ChangeTaskStatus(w http.ResponseWriter, r *http.Request, taskId int) {
	var body api.ChangeTaskStatusJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	err := s.Queries.ChangeTaskStatus(r.Context(), db.ChangeTaskStatusParams{
		Status: db.NullTaskStatus{
			TaskStatus: db.TaskStatus(body.Status),
			Valid:      true,
		},
		TaskID: int32(taskId),
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	jsonWrite(w, http.StatusOK, map[string]string{"message": "Статус задачи обновлён"})
}

func (s *TaskTrackerServer) DeleteTask(w http.ResponseWriter, r *http.Request, taskId int) {
	err := s.Queries.DeleteTask(r.Context(), int32(taskId))
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Member handlers ---

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
	_, err := s.Queries.AddMember(r.Context(), db.AddMemberParams{
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

func (s *TaskTrackerServer) KickUser(w http.ResponseWriter, r *http.Request, workspaceId uuid.UUID, userId uuid.UUID) {
	err := s.Queries.KickUser(r.Context(), userId)
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

// --- main ---

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
	defer conn.Close(ctx)

	queries := db.New(conn)

	serverImpl := &TaskTrackerServer{
		Queries: queries,
		DBConn:  conn,
	}

	r := chi.NewRouter()
	r.Use(requestLogger)
	r.Use(middleware.Recoverer)

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		api.HandlerFromMux(serverImpl, r)
	})

	r.Get("/frontend", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "frontend.html")
	})

	port := ":8080"
	fmt.Printf("Task Tracker api запущен на http://localhost%s\n", port)
	if err := http.ListenAndServe(port, r); err != nil {
		panic(err)
	}
}
