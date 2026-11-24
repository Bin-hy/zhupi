package httpapi

import (
	"net/http"

	"github.com/Bin-hy/zhupi/internal/domain"
)

func (s *Server) handleMeSites(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFrom(r.Context())
	if !ok {
		s.writeErr(w, r, domain.ErrUnauthorized)
		return
	}
	sites, err := s.admin.AccessibleSites(r.Context(), p)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sites": sites})
}

func (s *Server) handleAdminOverview(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFrom(r.Context())
	if !ok {
		s.writeErr(w, r, domain.ErrUnauthorized)
		return
	}
	overview, err := s.admin.Overview(r.Context(), p)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, overview)
}
