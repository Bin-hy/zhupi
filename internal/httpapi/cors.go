package httpapi

import (
	"net/http"
	"strings"

	"github.com/Bin-hy/go-campus/annotation-service/internal/service"
)

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := false
		if origin != "" {
			allowed = s.originAllowed(r, origin)
		}
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Site-Key")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Max-Age", "600")
			w.Header().Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			if origin != "" && !allowed {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) originAllowed(r *http.Request, origin string) bool {
	normalized, err := service.NormalizeOrigin(origin)
	if err != nil {
		return false
	}
	if strings.HasPrefix(r.URL.Path, "/api/v1/admin/") {
		for _, item := range s.cfg.AdminOrigins {
			if strings.EqualFold(item, normalized) {
				return true
			}
		}
		return false
	}
	siteKey := r.Header.Get("X-Site-Key")
	if siteKey == "" {
		siteKey = r.URL.Query().Get("siteKey")
	}
	if siteKey == "" {
		// 浏览器预检不带 X-Site-Key，只按「来源是否属于任一启用站点」放行；
		// 真正的请求会再次按 siteKey 与来源严格校验。
		if r.Method == http.MethodOptions {
			ok, err := s.store.OriginAllowedAnywhere(r.Context(), normalized)
			return err == nil && ok
		}
		return false
	}
	site, err := s.store.SiteByKey(r.Context(), siteKey)
	if err != nil || site.Status != "active" {
		return false
	}
	for _, item := range site.Origins {
		if strings.EqualFold(item, normalized) {
			return true
		}
	}
	return false
}
