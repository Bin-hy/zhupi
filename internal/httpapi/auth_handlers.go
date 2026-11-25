package httpapi

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/Bin-hy/zhupi/internal/domain"
	"github.com/Bin-hy/zhupi/internal/service"
)

func (s *Server) meta(r *http.Request) service.ClientMeta {
	return service.ClientMeta{UserAgent: r.UserAgent(), IP: s.clientIP(r)}
}

func (s *Server) handleGitHubStart(w http.ResponseWriter, r *http.Request) {
	site, err := s.sites.ResolveForLogin(r.Context(), r.URL.Query().Get("siteKey"))
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	redirectURL, err := s.auth.StartGitHub(r.Context(), site, r.URL.Query().Get("redirect"))
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (s *Server) handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if errCode := query.Get("error"); errCode != "" {
		s.writeErr(w, r, domain.Invalid("oauth_cancelled", "登录已取消："+errCode))
		return
	}
	if query.Get("code") == "" {
		s.writeErr(w, r, domain.Invalid("oauth_code_missing", "缺少授权码"))
		return
	}
	res, err := s.auth.FinishGitHub(r.Context(), query.Get("code"), query.Get("state"), s.meta(r))
	if err != nil {
		if s.redirectBindError(w, r, err) {
			return
		}
		s.writeErr(w, r, err)
		return
	}
	s.redirectAuthResult(w, r, res)
}

func (s *Server) handleOIDCStart(w http.ResponseWriter, r *http.Request) {
	providerID, ok := parseID(r.PathValue("providerID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的提供商 ID"))
		return
	}
	site, err := s.sites.ResolveForLogin(r.Context(), r.URL.Query().Get("siteKey"))
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	redirectURL, err := s.auth.StartOIDC(r.Context(), providerID, site, r.URL.Query().Get("redirect"))
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (s *Server) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	providerID, ok := parseID(r.PathValue("providerID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的提供商 ID"))
		return
	}
	query := r.URL.Query()
	if errCode := query.Get("error"); errCode != "" {
		s.writeErr(w, r, domain.Invalid("oauth_cancelled", "登录已取消："+errCode))
		return
	}
	if query.Get("code") == "" {
		s.writeErr(w, r, domain.Invalid("oauth_code_missing", "缺少授权码"))
		return
	}
	res, err := s.auth.FinishOIDC(r.Context(), providerID, query.Get("code"), query.Get("state"), s.meta(r))
	if err != nil {
		if s.redirectBindError(w, r, err) {
			return
		}
		s.writeErr(w, r, err)
		return
	}
	s.redirectAuthResult(w, r, res)
}

func (s *Server) redirectAuthResult(w http.ResponseWriter, r *http.Request, res service.LoginResult) {
	u, err := url.Parse(res.ReturnTo)
	if err != nil {
		s.writeErr(w, r, domain.ErrInternal)
		return
	}
	if res.Mode == domain.FlowBind {
		u.Fragment = "annot_bind=ok"
	} else {
		u.Fragment = "annot_token=" + url.QueryEscape(res.Token)
	}
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (s *Server) redirectBindError(w http.ResponseWriter, r *http.Request, err error) bool {
	var bindErr *service.BindError
	if !errors.As(err, &bindErr) {
		return false
	}
	target, parseErr := url.Parse(bindErr.Flow.ReturnTo)
	if parseErr != nil || target.Scheme == "" || target.Host == "" {
		return false
	}
	message := "绑定失败，请重试"
	if appErr, ok := domain.AsAppError(bindErr.Err); ok {
		message = appErr.Message
	}
	target.Fragment = "annot_bind=error&message=" + message
	http.Redirect(w, r, target.String(), http.StatusFound)
	return true
}

func (s *Server) handleBindGitHubStart(w http.ResponseWriter, r *http.Request) {
	site, ok := SiteFrom(r.Context())
	if !ok {
		s.writeErr(w, r, domain.ErrInternal)
		return
	}
	p, _ := PrincipalFrom(r.Context())
	startURL, err := s.auth.StartBindGitHub(r.Context(), p.User.ID, site, r.URL.Query().Get("redirect"))
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"redirectUrl": startURL})
}

func (s *Server) handleBindOIDCStart(w http.ResponseWriter, r *http.Request) {
	site, ok := SiteFrom(r.Context())
	if !ok {
		s.writeErr(w, r, domain.ErrInternal)
		return
	}
	providerID, ok := parseID(r.PathValue("providerID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的提供商 ID"))
		return
	}
	p, _ := PrincipalFrom(r.Context())
	startURL, err := s.auth.StartBindOIDC(r.Context(), p.User.ID, providerID, site, r.URL.Query().Get("redirect"))
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"redirectUrl": startURL})
}

type devLoginRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleDevLogin(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.DevLogin {
		s.writeErr(w, r, domain.NotFound("未开启调试登录"))
		return
	}
	var req devLoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	res, err := s.auth.DevLogin(r.Context(), req.Name, s.meta(r))
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token":     res.Token,
		"expiresAt": res.ExpiresAt,
		"user":      res.User,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	p, _ := PrincipalFrom(r.Context())
	sess, _ := SessionFrom(r.Context())
	if err := s.auth.Logout(r.Context(), p.User.ID, sess.ID); err != nil {
		s.writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFrom(r.Context())
	if !ok {
		s.writeErr(w, r, domain.ErrUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":                 p.User.ID,
		"displayName":        p.User.DisplayName,
		"avatarUrl":          p.User.AvatarURL,
		"isSuperadmin":       p.IsSuperadmin,
		"username":           p.LocalUsername,
		"mustChangePassword": p.MustChangePassword,
	})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	p, _ := PrincipalFrom(r.Context())
	sess, _ := SessionFrom(r.Context())
	if err := s.auth.ChangePassword(r.Context(), p.User.ID, sess.ID, req.CurrentPassword, req.NewPassword); err != nil {
		s.writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListIdentities(w http.ResponseWriter, r *http.Request) {
	p, _ := PrincipalFrom(r.Context())
	list, err := s.auth.ListIdentities(r.Context(), p.User.ID)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"identities": list})
}

func (s *Server) handleUnbindIdentity(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的身份 ID"))
		return
	}
	p, _ := PrincipalFrom(r.Context())
	if err := s.auth.UnbindIdentity(r.Context(), p.User.ID, id); err != nil {
		s.writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	p, _ := PrincipalFrom(r.Context())
	current, _ := SessionFrom(r.Context())
	list, err := s.auth.ListSessions(r.Context(), p.User.ID)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"sessions":         list,
		"currentSessionId": current.ID,
	})
}

func (s *Server) handleRevokeSession(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的会话 ID"))
		return
	}
	p, _ := PrincipalFrom(r.Context())
	if err := s.auth.RevokeSession(r.Context(), p.User.ID, id); err != nil {
		s.writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	p, _ := PrincipalFrom(r.Context())
	if err := s.auth.RevokeAllSessions(r.Context(), p.User.ID); err != nil {
		s.writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
