package httpserver

import (
	"errors"
	"net/http"

	"github.com/thomas666-beast/marketplace/internal/catalog"
)

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	locale := localeFromRequest(r)

	tree, err := s.categories.TreeActive(r.Context())
	if err != nil {
		s.logger.Error("list categories failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"items": toCategoryTree(tree, locale),
	})
}

func (s *Server) handleGetCategory(w http.ResponseWriter, r *http.Request) {
	locale := localeFromRequest(r)
	slug := r.PathValue("slug")

	c, err := s.categories.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, catalog.ErrCategoryNotFound) {
			s.errorResponse(w, r, http.StatusNotFound, "category_not_found", "catalog.category_not_found")
			return
		}
		s.logger.Error("get category failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	s.writeJSON(w, http.StatusOK, toCategoryResponse(c, locale))
}
