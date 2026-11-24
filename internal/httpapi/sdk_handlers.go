package httpapi

import (
	"net/http"
	"strings"

	"github.com/Bin-hy/zhupi/internal/sdkassets"
)

func (s *Server) handleSDKFile(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.PathValue("name"), "/")
	if name == "" || strings.Contains(name, "..") {
		http.NotFound(w, r)
		return
	}
	data, err := sdkassets.Read(name)
	if err != nil {
		http.Error(w, "SDK 尚未构建，请先执行 make sdk", http.StatusNotFound)
		return
	}
	switch {
	case strings.HasSuffix(name, ".js"):
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case strings.HasSuffix(name, ".map"):
		w.Header().Set("Content-Type", "application/json")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(data)
}
