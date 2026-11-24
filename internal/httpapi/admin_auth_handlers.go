package httpapi

import (
	"net/http"

	"github.com/Bin-hy/zhupi/internal/domain"
)

func (s *Server) handleAdminProviders(w http.ResponseWriter, r *http.Request) {
	out, err := s.auth.AdminProviders(r.Context())
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type localLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLocalLogin(w http.ResponseWriter, r *http.Request) {
	var req localLoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	res, err := s.auth.LocalLogin(r.Context(), req.Username, req.Password, s.meta(r))
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token":              res.Token,
		"expiresAt":          res.ExpiresAt,
		"user":               res.User,
		"mustChangePassword": res.MustChangePassword,
	})
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
