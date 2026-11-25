package httpapi

import (
	"net/http"

	"github.com/Bin-hy/zhupi/internal/domain"
	"github.com/Bin-hy/zhupi/internal/service"
)

func (s *Server) handleAdminListLocalAdmins(w http.ResponseWriter, r *http.Request) {
	p, _ := PrincipalFrom(r.Context())
	list, err := s.admin.ListLocalAdmins(r.Context(), p)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"admins": list})
}

func (s *Server) handleAdminCreateLocalAdmin(w http.ResponseWriter, r *http.Request) {
	var req service.CreateLocalAdminInput
	if !decodeJSON(w, r, &req) {
		return
	}
	p, _ := PrincipalFrom(r.Context())
	res, err := s.admin.CreateLocalAdmin(r.Context(), p, req)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	s.logger.Info("admin account created",
		"actorId", p.User.ID, "userId", res.Account.UserID, "username", res.Account.Username)
	writeJSON(w, http.StatusCreated, res)
}

func (s *Server) handleAdminOpenLocalCredential(w http.ResponseWriter, r *http.Request) {
	var req service.OpenLocalCredentialInput
	if !decodeJSON(w, r, &req) {
		return
	}
	p, _ := PrincipalFrom(r.Context())
	res, err := s.admin.OpenLocalCredential(r.Context(), p, req)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	s.logger.Info("local credential opened",
		"actorId", p.User.ID, "userId", res.Account.UserID, "username", res.Account.Username)
	writeJSON(w, http.StatusCreated, res)
}

type resetLocalPasswordRequest struct {
	Password string `json:"password"`
}

func (s *Server) handleAdminResetLocalPassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseID(r.PathValue("userID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的用户 ID"))
		return
	}
	var req resetLocalPasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	p, _ := PrincipalFrom(r.Context())
	generated, err := s.admin.ResetLocalAdminPassword(r.Context(), p, userID, req.Password)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	s.logger.Info("local admin password reset", "actorId", p.User.ID, "userId", userID)
	writeJSON(w, http.StatusOK, map[string]string{"generatedPassword": generated})
}

type setSuperadminRequest struct {
	IsSuperadmin *bool `json:"isSuperadmin"`
}

func (s *Server) handleAdminSetSuperadmin(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseID(r.PathValue("userID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的用户 ID"))
		return
	}
	var req setSuperadminRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.IsSuperadmin == nil {
		s.writeErr(w, r, domain.Invalid("validation_error", "缺少 isSuperadmin"))
		return
	}
	p, _ := PrincipalFrom(r.Context())
	if err := s.admin.SetSuperadmin(r.Context(), p, userID, *req.IsSuperadmin); err != nil {
		s.writeErr(w, r, err)
		return
	}
	s.logger.Info("superadmin role changed",
		"actorId", p.User.ID, "userId", userID, "isSuperadmin", *req.IsSuperadmin)
	w.WriteHeader(http.StatusNoContent)
}
