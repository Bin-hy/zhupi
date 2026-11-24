package httpapi

import (
	"net/http"

	"github.com/Bin-hy/go-campus/annotation-service/internal/domain"
)

func (s *Server) handleAdminProviders(w http.ResponseWriter, r *http.Request) {
	out, err := s.auth.AdminProviders(r.Context())
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAdminGitHubStart(w http.ResponseWriter, r *http.Request) {
	target, err := s.auth.StartAdminGitHub(r.Context(), r.URL.Query().Get("redirect"))
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func (s *Server) handleAdminOIDCStart(w http.ResponseWriter, r *http.Request) {
	providerID, ok := parseID(r.PathValue("providerID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的提供商 ID"))
		return
	}
	target, err := s.auth.StartAdminOIDC(r.Context(), providerID, r.URL.Query().Get("redirect"))
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}
