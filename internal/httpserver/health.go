package httpserver

import "net/http"

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    locale := localeFromRequest(r)

    status := map[string]any{
        "status":  "ok",
        "message": s.i18n.Translate(locale, "health.ok"),
    }

    if err := s.db.Pool.Ping(r.Context()); err != nil {
        s.logger.Error("db ping failed", "err", err)
        s.writeJSON(w, http.StatusServiceUnavailable, map[string]any{
            "status": "degraded",
            "error":  err.Error(),
        })
        return
    }
    status["db"] = s.i18n.Translate(locale, "health.db_ok")

    s.writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleWelcome(w http.ResponseWriter, r *http.Request) {
    locale := localeFromRequest(r)
    s.writeJSON(w, http.StatusOK, map[string]string{
        "message": s.i18n.Translate(locale, "welcome"),
    })
}
