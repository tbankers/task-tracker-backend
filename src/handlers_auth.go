package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/tbankers/task-tracker-backend/api"
	db "github.com/tbankers/task-tracker-backend/db"
)

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

	jsonWrite(w, http.StatusCreated, map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"user_id":    userID.String(),
			"email":      string(body.Email),
			"username":   body.Username,
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
	resetLink := fmt.Sprintf("%s/#/reset-password?token=%s", getEnv("FRONTEND_URL", "http://localhost:3000"), resetToken)
	emailBody := passwordResetEmailHTML(user.Email, resetLink)

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
		_ = s.Queries.DeletePasswordResetToken(r.Context(), body.Token)
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
	_ = s.Queries.DeletePasswordResetToken(r.Context(), body.Token)
	jsonWrite(w, http.StatusOK, map[string]string{
		"message": "Пароль успешно изменён",
	})
}

func (s *TaskTrackerServer) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Токен не указан")
		return
	}
	verificationToken, err := s.Queries.GetEmailVerificationToken(r.Context(), token)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Неверный или просроченный токен")
		return
	}
	if time.Now().After(verificationToken.ExpiresAt.Time) {
		_ = s.Queries.DeleteEmailVerificationToken(r.Context(), token)
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Токен просрочен")
		return
	}
	err = s.Queries.SetEmailVerified(r.Context(), verificationToken.UserID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	_ = s.Queries.DeleteEmailVerificationToken(r.Context(), token)
	jsonWrite(w, http.StatusOK, map[string]string{
		"message": "Email подтверждён",
	})
}

func (s *TaskTrackerServer) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, http.StatusBadRequest, "BAD_REQUEST", "Невалидный JSON")
		return
	}
	user, err := s.Queries.GetUserByEmail(r.Context(), body.Email)
	if err != nil {
		jsonWrite(w, http.StatusOK, map[string]string{
			"message": "Если пользователь с таким email существует, письмо отправлено",
		})
		return
	}
	if user.EmailVerified {
		jsonWrite(w, http.StatusOK, map[string]string{
			"message": "Email уже подтверждён",
		})
		return
	}
	_ = s.Queries.DeleteEmailVerificationTokensByUserID(r.Context(), user.UserID)
	verificationToken, err := generateResetToken()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Ошибка генерации токена")
		return
	}
	expiresAt := pgtype.Timestamp{Time: time.Now().Add(24 * time.Hour), Valid: true}
	_, err = s.Queries.CreateEmailVerificationToken(r.Context(), db.CreateEmailVerificationTokenParams{
		UserID:    user.UserID,
		Token:     verificationToken,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	verifyLink := fmt.Sprintf("%s/#/verify-email?token=%s", getEnv("FRONTEND_URL", "http://localhost:3000"), verificationToken)
	emailBody := verificationEmailHTML(user.Username, verifyLink)
	if err := sendEmail(user.Email, "Подтвердите email — Task Tracker", emailBody); err != nil {
		fmt.Printf("[EMAIL ERROR] %v\n", err)
		sendError(w, http.StatusInternalServerError, "EMAIL_ERROR", err.Error())
		return
	}
	jsonWrite(w, http.StatusOK, map[string]string{
		"message": "Письмо с подтверждением отправлено",
	})
}
