package utils

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

func ParseJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return fmt.Errorf("missing request body")
	}
	return json.NewDecoder(r.Body).Decode(v)
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func WriteMessage(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"message": msg})
}

func WriteError(w http.ResponseWriter, status int, err error) {
	WriteJSON(w, status, map[string]string{"error": err.Error()})
}

func ParseTime(str *string) *time.Time {
	if str == nil || *str == "" {
		return nil
	}

	s := *str
	if !strings.HasSuffix(s, "Z") {
		s += "Z"
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		slog.Error("time parse error", "error", err, "input", s)
		return nil
	}
	return &t
}
