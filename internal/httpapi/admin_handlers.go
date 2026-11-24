package httpapi

import (
	"net/http"
	"strings"

	"github.com/Bin-hy/go-campus/annotation-service/internal/domain"
	"github.com/Bin-hy/go-campus/annotation-service/internal/service"
)

func (s *Server) handleAdminListSites(w http.ResponseWriter, r *http.Request) {
	cursor := queryInt64(r, "cursor", 0)
	limit := int(queryInt64(r, "limit", 50))
	list, next, err := s.sites.List(r.Context(), cursor, limit)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sites": list, "nextCursor": next})
}

func (s *Server) handleAdminCreateSite(w http.ResponseWriter, r *http.Request) {
	var in service.CreateSiteInput
	if !decodeJSON(w, r, &in) {
		return
	}
	site, err := s.sites.Create(r.Context(), in)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, site)
}

func (s *Server) handleAdminUpdateSite(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的站点 ID"))
		return
	}
	var in service.UpdateSiteInput
	if !decodeJSON(w, r, &in) {
		return
	}
	site, err := s.sites.Update(r.Context(), id, in)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, site)
}

func (s *Server) handleAdminRotateKey(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的站点 ID"))
		return
	}
	key, err := s.sites.RotateKey(r.Context(), id)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"siteKey": key})
}

func (s *Server) handleAdminListAdmins(w http.ResponseWriter, r *http.Request) {
	siteID, ok := parseID(r.PathValue("id"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的站点 ID"))
		return
	}
	p, _ := PrincipalFrom(r.Context())
	list, err := s.admin.ListSiteAdmins(r.Context(), p, siteID)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"admins": list})
}

type addAdminRequest struct {
	UserID int64 `json:"userId"`
}

func (s *Server) handleAdminAddAdmin(w http.ResponseWriter, r *http.Request) {
	siteID, ok := parseID(r.PathValue("id"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的站点 ID"))
		return
	}
	var req addAdminRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.UserID <= 0 {
		s.writeErr(w, r, domain.Invalid("validation_error", "缺少 userId"))
		return
	}
	p, _ := PrincipalFrom(r.Context())
	if err := s.admin.AddSiteAdmin(r.Context(), p, siteID, req.UserID); err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"siteId": siteID, "userId": req.UserID})
}

func (s *Server) handleAdminRemoveAdmin(w http.ResponseWriter, r *http.Request) {
	siteID, ok := parseID(r.PathValue("id"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的站点 ID"))
		return
	}
	userID, ok := parseID(r.PathValue("userID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的用户 ID"))
		return
	}
	p, _ := PrincipalFrom(r.Context())
	if err := s.admin.RemoveSiteAdmin(r.Context(), p, siteID, userID); err != nil {
		s.writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	p, _ := PrincipalFrom(r.Context())
	cursor := queryInt64(r, "cursor", 0)
	limit := int(queryInt64(r, "limit", 50))
	list, next, err := s.admin.ListUsers(r.Context(), p, cursor, limit)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": list, "nextCursor": next})
}

func (s *Server) handleAdminGlobalBan(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的用户 ID"))
		return
	}
	p, _ := PrincipalFrom(r.Context())
	if err := s.admin.GlobalBan(r.Context(), p, id); err != nil {
		s.writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminGlobalUnban(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的用户 ID"))
		return
	}
	p, _ := PrincipalFrom(r.Context())
	if err := s.admin.GlobalUnban(r.Context(), p, id); err != nil {
		s.writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminQueue(w http.ResponseWriter, r *http.Request) {
	siteID, ok := parseID(r.PathValue("siteID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的站点 ID"))
		return
	}
	p, err := s.requireModerator(r, siteID)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	status := domain.ModerationStatus(r.URL.Query().Get("status"))
	if status == "" {
		status = domain.ModerationPending
	}
	queue, err := s.moderation.Queue(r.Context(), p, siteID, status,
		queryInt64(r, "annotationCursor", 0), queryInt64(r, "commentCursor", 0),
		int(queryInt64(r, "limit", 50)))
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, queue)
}

type moderationRequest struct {
	Action string `json:"action"`
}

func (s *Server) handleAdminModerateAnnotation(w http.ResponseWriter, r *http.Request) {
	siteID, ok := parseID(r.PathValue("siteID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的站点 ID"))
		return
	}
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的批注 ID"))
		return
	}
	p, err := s.requireModerator(r, siteID)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	var req moderationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	action := service.ModerationAction(req.Action)
	if err := s.moderation.ModerateAnnotation(r.Context(), p, siteID, id, action); err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"action": req.Action})
}

func (s *Server) handleAdminModerateComment(w http.ResponseWriter, r *http.Request) {
	siteID, ok := parseID(r.PathValue("siteID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的站点 ID"))
		return
	}
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的评论 ID"))
		return
	}
	p, err := s.requireModerator(r, siteID)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	var req moderationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	action := service.ModerationAction(req.Action)
	if err := s.moderation.ModerateComment(r.Context(), p, siteID, id, action); err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"action": req.Action})
}

type banRequest struct {
	Reason string `json:"reason"`
}

func (s *Server) handleAdminBanUser(w http.ResponseWriter, r *http.Request) {
	siteID, ok := parseID(r.PathValue("siteID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的站点 ID"))
		return
	}
	userID, ok := parseID(r.PathValue("userID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的用户 ID"))
		return
	}
	p, err := s.requireModerator(r, siteID)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	var req banRequest
	if r.ContentLength > 0 {
		if !decodeJSON(w, r, &req) {
			return
		}
	}
	if err := s.moderation.BanUser(r.Context(), p, siteID, userID, req.Reason); err != nil {
		s.writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminUnbanUser(w http.ResponseWriter, r *http.Request) {
	siteID, ok := parseID(r.PathValue("siteID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的站点 ID"))
		return
	}
	userID, ok := parseID(r.PathValue("userID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的用户 ID"))
		return
	}
	p, err := s.requireModerator(r, siteID)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	if err := s.moderation.UnbanUser(r.Context(), p, siteID, userID); err != nil {
		s.writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminListProviders(w http.ResponseWriter, r *http.Request) {
	siteID, ok := parseID(r.PathValue("siteID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的站点 ID"))
		return
	}
	if _, err := s.requireModerator(r, siteID); err != nil {
		s.writeErr(w, r, err)
		return
	}
	list, err := s.store.ListProvidersBySite(r.Context(), siteID, false)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": list})
}

type providerRequest struct {
	Name                string   `json:"name"`
	Issuer              string   `json:"issuer"`
	ClientID            string   `json:"clientId"`
	ClientSecret        string   `json:"clientSecret"`
	Scopes              []string `json:"scopes"`
	AllowedEmailDomains []string `json:"allowedEmailDomains"`
	Enabled             *bool    `json:"enabled"`
}

func (s *Server) handleAdminCreateProvider(w http.ResponseWriter, r *http.Request) {
	siteID, ok := parseID(r.PathValue("siteID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的站点 ID"))
		return
	}
	if _, err := s.requireModerator(r, siteID); err != nil {
		s.writeErr(w, r, err)
		return
	}
	var req providerRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := validateProviderInput(req, true); err != nil {
		s.writeErr(w, r, err)
		return
	}
	issuer := strings.TrimRight(strings.TrimSpace(req.Issuer), "/")
	if err := s.auth.ValidateIssuer(r.Context(), issuer); err != nil {
		s.writeErr(w, r, domain.Invalid("oidc_discovery_failed", "OIDC Discovery 校验失败："+err.Error()))
		return
	}
	cipher, err := s.auth.EncryptSecret(req.ClientSecret)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	p, err := s.store.CreateProvider(r.Context(), domain.OIDCProvider{
		SiteID: siteID, Name: strings.TrimSpace(req.Name), Issuer: issuer, ClientID: strings.TrimSpace(req.ClientID),
		ClientSecretCipher: cipher, Scopes: scopesOrDefault(req.Scopes),
		AllowedEmailDomains: normalizeDomains(req.AllowedEmailDomains), Enabled: enabled,
	})
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleAdminUpdateProvider(w http.ResponseWriter, r *http.Request) {
	siteID, ok := parseID(r.PathValue("siteID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的站点 ID"))
		return
	}
	providerID, ok := parseID(r.PathValue("providerID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的提供商 ID"))
		return
	}
	if _, err := s.requireModerator(r, siteID); err != nil {
		s.writeErr(w, r, err)
		return
	}
	existing, err := s.store.ProviderByID(r.Context(), siteID, providerID)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	var req providerRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) != "" {
		existing.Name = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.Issuer) != "" {
		existing.Issuer = strings.TrimRight(strings.TrimSpace(req.Issuer), "/")
	}
	if strings.TrimSpace(req.ClientID) != "" {
		existing.ClientID = strings.TrimSpace(req.ClientID)
	}
	if req.Scopes != nil {
		existing.Scopes = scopesOrDefault(req.Scopes)
	}
	if req.AllowedEmailDomains != nil {
		existing.AllowedEmailDomains = normalizeDomains(req.AllowedEmailDomains)
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if strings.TrimSpace(req.ClientSecret) != "" {
		cipher, err := s.auth.EncryptSecret(req.ClientSecret)
		if err != nil {
			s.writeErr(w, r, err)
			return
		}
		existing.ClientSecretCipher = cipher
	}
	if err := s.auth.ValidateIssuer(r.Context(), existing.Issuer); err != nil {
		s.writeErr(w, r, domain.Invalid("oidc_discovery_failed", "OIDC Discovery 校验失败："+err.Error()))
		return
	}
	if err := s.store.UpdateProvider(r.Context(), existing); err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) handleAdminDeleteProvider(w http.ResponseWriter, r *http.Request) {
	siteID, ok := parseID(r.PathValue("siteID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的站点 ID"))
		return
	}
	providerID, ok := parseID(r.PathValue("providerID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的提供商 ID"))
		return
	}
	if _, err := s.requireModerator(r, siteID); err != nil {
		s.writeErr(w, r, err)
		return
	}
	if err := s.store.DeleteProvider(r.Context(), siteID, providerID); err != nil {
		s.writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateProviderInput(req providerRequest, requireSecret bool) error {
	if strings.TrimSpace(req.Name) == "" || len(strings.TrimSpace(req.Name)) > 64 {
		return domain.Invalid("validation_error", "提供商名称需为 1–64 字符")
	}
	if strings.TrimSpace(req.Issuer) == "" {
		return domain.Invalid("validation_error", "issuer 不能为空")
	}
	if strings.TrimSpace(req.ClientID) == "" {
		return domain.Invalid("validation_error", "clientId 不能为空")
	}
	if requireSecret && strings.TrimSpace(req.ClientSecret) == "" {
		return domain.Invalid("validation_error", "clientSecret 不能为空")
	}
	return nil
}

func scopesOrDefault(scopes []string) []string {
	if len(scopes) == 0 {
		return []string{"openid", "email", "profile"}
	}
	out := make([]string, 0, len(scopes))
	for _, s := range scopes {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return []string{"openid", "email", "profile"}
	}
	return out
}

func normalizeDomains(domains []string) []string {
	out := make([]string, 0, len(domains))
	for _, d := range domains {
		if d = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(d, "@"))); d != "" {
			out = append(out, d)
		}
	}
	return out
}
