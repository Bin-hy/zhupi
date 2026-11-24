package httpapi

import (
	"net/http"

	"github.com/Bin-hy/go-campus/annotation-service/internal/domain"
	"github.com/Bin-hy/go-campus/annotation-service/internal/sdkassets"
)

func (s *Server) handleSDKJS(w http.ResponseWriter, r *http.Request) {
	s.serveSDKFile(w, "annotation-sdk.js")
}

func (s *Server) handleSDKESM(w http.ResponseWriter, r *http.Request) {
	s.serveSDKFile(w, "annotation-sdk.esm.js")
}

func (s *Server) serveSDKFile(w http.ResponseWriter, name string) {
	data, err := sdkassets.Read(name)
	if err != nil {
		http.Error(w, "SDK 尚未构建，请先执行 make sdk", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(data)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "db_unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

type authProviderView struct {
	ID     int64  `json:"id"`
	Type   string `json:"type"`
	Name   string `json:"name"`
	Issuer string `json:"issuer,omitempty"`
}

func (s *Server) handleSiteConfig(w http.ResponseWriter, r *http.Request) {
	site, ok := SiteFrom(r.Context())
	if !ok {
		s.writeErr(w, r, domain.ErrInternal)
		return
	}
	providers := make([]authProviderView, 0, 2)
	if s.cfg.GitHubConfigured() {
		providers = append(providers, authProviderView{ID: 0, Type: "github", Name: "GitHub"})
	}
	list, err := s.store.ListProvidersBySite(r.Context(), site.ID, true)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	for _, p := range list {
		providers = append(providers, authProviderView{ID: p.ID, Type: "oidc", Name: p.Name, Issuer: p.Issuer})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"siteKey":        site.Key,
		"displayName":    site.DisplayName,
		"themeColor":     site.ThemeColor,
		"locale":         site.DefaultLocale,
		"moderationMode": site.ModerationMode,
		"providers":      providers,
	})
}
