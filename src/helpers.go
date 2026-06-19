package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func sendError(w http.ResponseWriter, statusCode int, code string, msg string) {
	fmt.Printf("[ERROR] %d %s: %s\n", statusCode, code, msg)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    code,
		"message": msg,
	})
}

func jsonWrite(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
