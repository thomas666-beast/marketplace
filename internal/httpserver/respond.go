package httpserver

import (
	"encoding/json"
	"net/http"
)

type errorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) errorResponse(w http.ResponseWriter, r *http.Request, status int, errorKey, messageKey string) {
	locale := localeFromRequest(r)
	body := errorBody{
		Error:   errorKey,
		Message: s.i18n.Translate(locale, messageKey),
	}
	s.writeJSON(w, status, body)
}

func localeFromRequest(r *http.Request) string {
	header := r.Header.Get("Accept-Language")
	if header == "" {
		return "en"
	}
	for i := 0; i < len(header); i++ {
		if header[i] == ',' || header[i] == ';' {
			header = header[:i]
			break
		}
	}
	if header == "" {
		return "en"
	}
	return header
}
