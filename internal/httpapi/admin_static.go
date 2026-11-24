package httpapi

import (
	"net/http"
	"path"
	"strings"

	"github.com/Bin-hy/go-campus/annotation-service/internal/adminassets"
)

func (s *Server) handleAdminRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin/", http.StatusFound)
}

func (s *Server) handleAdminSPA(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.PathValue("path"), "/")
	if name == "" || path.Ext(name) == "" {
		s.serveAdminIndex(w)
		return
	}
	data, err := adminassets.Read(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentTypeFor(name))
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(data)
}

func (s *Server) serveAdminIndex(w http.ResponseWriter) {
	data, err := adminassets.Read("index.html")
	if err != nil {
		http.Error(w, "管理后台尚未构建，请先执行 make admin", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}

func contentTypeFor(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".js", ".mjs":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".html":
		return "text/html; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".json", ".map":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
