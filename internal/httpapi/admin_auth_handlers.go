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

func (s *Server) handleAdminIdentities(w http.ResponseWriter, r *http.Request) {
	p, _ := PrincipalFrom(r.Context())
	out, err := s.auth.AdminBindableIdentities(r.Context(), p.User.ID)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAdminBindGitHubStart(w http.ResponseWriter, r *http.Request) {
	p, _ := PrincipalFrom(r.Context())
	redirectURL, err := s.auth.StartAdminBindGitHub(r.Context(), p.User.ID)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"redirectUrl": redirectURL})
}

func (s *Server) handleAdminBindOIDCStart(w http.ResponseWriter, r *http.Request) {
	providerID, ok := parseID(r.PathValue("providerID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的提供商 ID"))
		return
	}
	p, _ := PrincipalFrom(r.Context())
	redirectURL, err := s.auth.StartAdminBindOIDC(r.Context(), p.User.ID, providerID)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"redirectUrl": redirectURL})
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
