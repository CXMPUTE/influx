package main

import (
	"net/http"
)

func RotateHandler(store *TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		newToken, err := store.Rotate()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = writeJSON(w, map[string]any{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = writeJSON(w, map[string]any{
			"rotated":   true,
			"new_token": newToken,
		})
	}
}
