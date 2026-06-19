package main

import (
	"fmt"
	"os"
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

type contextKey string

const (
	contextKeyUserID contextKey = "user_id"
	contextKeyEmail  contextKey = "email"
)

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}
	jwtSecret = []byte(secret)

	port := 587
	_, _ = fmt.Sscanf(os.Getenv("SMTP_PORT"), "%d", &port)
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
