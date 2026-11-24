package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Bin-hy/go-campus/annotation-service/internal/domain"
)

type ctxKey int

const (
	siteCtx ctxKey = iota
	principalCtx
	sessionCtx
	requestIDCtx
)

func SiteFrom(ctx context.Context) (domain.Site, bool) {
	site, ok := ctx.Value(siteCtx).(domain.Site)
	return site, ok
}

func PrincipalFrom(ctx context.Context) (domain.Principal, bool) {
	p, ok := ctx.Value(principalCtx).(domain.Principal)
	return p, ok
}

func SessionFrom(ctx context.Context) (domain.Session, bool) {
	s, ok := ctx.Value(sessionCtx).(domain.Session)
	return s, ok
}

func requestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDCtx).(string)
	return id
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write json failed", "err", err)
	}
}

func (s *Server) writeErr(w http.ResponseWriter, r *http.Request, err error) {
	appErr, ok := domain.AsAppError(err)
	if !ok {
		s.logger.Error("unhandled error", "path", r.URL.Path, "err", err, "requestId", requestIDFrom(r.Context()))
		appErr = domain.ErrInternal
	}
	if appErr.Status >= 500 {
		s.logger.Error("server error", "path", r.URL.Path, "code", appErr.Code, "err", appErr.Err, "requestId", requestIDFrom(r.Context()))
	}
	writeJSON(w, appErr.Status, map[string]any{
		"error": map[string]string{
			"code":      appErr.Code,
			"message":   appErr.Message,
			"requestId": requestIDFrom(r.Context()),
		},
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"code":      "validation_error",
				"message":   "请求体格式不正确或过大",
				"requestId": requestIDFrom(r.Context()),
			},
		})
		return false
	}
	return true
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if len(header) <= len("bearer ") || !strings.EqualFold(header[:len("bearer ")], "bearer ") {
		return ""
	}
	return strings.TrimSpace(header[len("bearer "):])
}

func (s *Server) clientIP(r *http.Request) string {
	if s.cfg.TrustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if first := strings.TrimSpace(strings.Split(xff, ",")[0]); first != "" {
				return first
			}
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func (s *Server) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 8)
		_, _ = rand.Read(buf)
		id := hex.EncodeToString(buf)
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDCtx, id)))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (s *Server) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"durationMs", time.Since(start).Milliseconds(),
			"ip", s.clientIP(r),
			"requestId", requestIDFrom(r.Context()),
		)
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.Error("panic", "path", r.URL.Path, "panic", rec, "requestId", requestIDFrom(r.Context()))
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"error": map[string]string{
						"code":      "internal_error",
						"message":   "服务器内部错误",
						"requestId": requestIDFrom(r.Context()),
					},
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) bodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPatch || r.Method == http.MethodPut {
			r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
		}
		next.ServeHTTP(w, r)
	})
}
