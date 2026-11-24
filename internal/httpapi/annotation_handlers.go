package httpapi

import (
	"net/http"
	"strconv"

	"github.com/Bin-hy/zhupi/internal/domain"
	"github.com/Bin-hy/zhupi/internal/service"
)

type createAnnotationRequest struct {
	PageKey   string        `json:"pageKey"`
	PageTitle string        `json:"pageTitle"`
	Anchor    domain.Anchor `json:"anchor"`
	Content   string        `json:"content"`
}

type patchAnnotationRequest struct {
	Status *string        `json:"status"`
	Anchor *domain.Anchor `json:"anchor"`
}

type commentRequest struct {
	Content string `json:"content"`
}

func (s *Server) handleListAnnotations(w http.ResponseWriter, r *http.Request) {
	site, _ := SiteFrom(r.Context())
	p, _ := PrincipalFrom(r.Context())
	pageKey := r.URL.Query().Get("page")
	filter := domain.ListFilter(r.URL.Query().Get("filter"))
	sort := domain.AnnotationSort(r.URL.Query().Get("sort"))
	cursor := queryInt64(r, "cursor", 0)
	limit := int(queryInt64(r, "limit", 100))
	result, err := s.annotations.ListPage(r.Context(), p, site, service.ListInput{
		PageKey: pageKey,
		Filter:  filter,
		Sort:    sort,
		Cursor:  cursor,
		Limit:   limit,
	})
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleCreateAnnotation(w http.ResponseWriter, r *http.Request) {
	site, _ := SiteFrom(r.Context())
	p, _ := PrincipalFrom(r.Context())
	var req createAnnotationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	a, err := s.annotations.Create(r.Context(), p, site, service.CreateAnnotationInput{
		PageKey:   req.PageKey,
		PageTitle: req.PageTitle,
		Anchor:    req.Anchor,
		Content:   req.Content,
	})
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (s *Server) handlePatchAnnotation(w http.ResponseWriter, r *http.Request) {
	site, _ := SiteFrom(r.Context())
	p, _ := PrincipalFrom(r.Context())
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的批注 ID"))
		return
	}
	var req patchAnnotationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	in := service.UpdateAnnotationInput{Anchor: req.Anchor}
	if req.Status != nil {
		status := domain.AnnotationStatus(*req.Status)
		in.Status = &status
	}
	updated, err := s.annotations.Update(r.Context(), p, site.ID, id, in)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteAnnotation(w http.ResponseWriter, r *http.Request) {
	site, _ := SiteFrom(r.Context())
	p, _ := PrincipalFrom(r.Context())
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的批注 ID"))
		return
	}
	if err := s.annotations.Delete(r.Context(), p, site.ID, id); err != nil {
		s.writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateComment(w http.ResponseWriter, r *http.Request) {
	site, _ := SiteFrom(r.Context())
	p, _ := PrincipalFrom(r.Context())
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的批注 ID"))
		return
	}
	var req commentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	c, err := s.annotations.AddComment(r.Context(), p, site, id, req.Content)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleDeleteComment(w http.ResponseWriter, r *http.Request) {
	site, _ := SiteFrom(r.Context())
	p, _ := PrincipalFrom(r.Context())
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的评论 ID"))
		return
	}
	if err := s.annotations.DeleteComment(r.Context(), p, site.ID, id); err != nil {
		s.writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func queryInt64(r *http.Request, key string, fallback int64) int64 {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v < 0 {
		return fallback
	}
	return v
}
