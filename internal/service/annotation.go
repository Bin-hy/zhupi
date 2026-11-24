package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/Bin-hy/go-campus/annotation-service/internal/domain"
	"github.com/Bin-hy/go-campus/annotation-service/internal/store"
)

type AnnotationService struct {
	annotations store.AnnotationRepo
	comments    store.CommentRepo
}

func NewAnnotationService(annotations store.AnnotationRepo, comments store.CommentRepo) *AnnotationService {
	return &AnnotationService{annotations: annotations, comments: comments}
}

type CreateAnnotationInput struct {
	PageKey   string
	PageTitle string
	Anchor    domain.Anchor
	Content   string
}

type ListInput struct {
	PageKey string
	Filter  domain.ListFilter
	Cursor  int64
	Limit   int
}

type UpdateAnnotationInput struct {
	Status *domain.AnnotationStatus
	Anchor *domain.Anchor
}

type PageResult struct {
	Annotations []domain.Annotation `json:"annotations"`
	NextCursor  int64               `json:"nextCursor"`
	Total       int                 `json:"total"`
}

func (s *AnnotationService) ListPage(ctx context.Context, p domain.Principal, site domain.Site, in ListInput) (PageResult, error) {
	pageKey := strings.TrimSpace(in.PageKey)
	if pageKey == "" || len(pageKey) > 512 {
		return PageResult{}, domain.Invalid("validation_error", "pageKey 需为 1–512 字符")
	}
	filter := in.Filter
	if filter == "" {
		filter = domain.ListAll
	}
	if !filter.Valid() {
		return PageResult{}, domain.Invalid("invalid_filter", "不支持的筛选条件")
	}
	if (filter == domain.ListMine || filter == domain.ListPending) && p.User.ID == 0 {
		return PageResult{}, domain.Unauthorized("该筛选需要登录")
	}
	include := p.IsSuperadmin || p.IsSiteAdmin
	list, next, err := s.annotations.ListAnnotations(ctx, store.AnnotationQuery{
		SiteID:            site.ID,
		PageKey:           pageKey,
		ViewerID:          p.User.ID,
		Filter:            filter,
		IncludeUnapproved: include,
		Cursor:            in.Cursor,
		Limit:             in.Limit,
	})
	if err != nil {
		return PageResult{}, err
	}
	total, err := s.annotations.CountPageAnnotations(ctx, site.ID, pageKey, p.User.ID, include)
	if err != nil {
		return PageResult{}, err
	}
	return PageResult{Annotations: list, NextCursor: next, Total: total}, nil
}

func (s *AnnotationService) Create(ctx context.Context, p domain.Principal, site domain.Site, in CreateAnnotationInput) (domain.Annotation, error) {
	if p.User.ID == 0 {
		return domain.Annotation{}, domain.ErrUnauthorized
	}
	pageKey := strings.TrimSpace(in.PageKey)
	if pageKey == "" || len(pageKey) > 512 {
		return domain.Annotation{}, domain.Invalid("validation_error", "pageKey 需为 1–512 字符")
	}
	content := strings.TrimSpace(in.Content)
	if !withinRunes(content, 1, 4000) {
		return domain.Annotation{}, domain.Invalid("validation_error", "批注内容需为 1–4000 字")
	}
	if err := validateAnchor(in.Anchor); err != nil {
		return domain.Annotation{}, err
	}
	if in.Anchor.Quote.Type == "" {
		in.Anchor.Quote.Type = "TextQuoteSelector"
	}

	moderation := domain.ModerationApproved
	if site.ModerationMode == domain.ModerationPre {
		moderation = domain.ModerationPending
	}
	a := domain.Annotation{
		SiteID:     site.ID,
		PageKey:    pageKey,
		PageTitle:  truncate(strings.TrimSpace(in.PageTitle), 200),
		Anchor:     in.Anchor,
		QuoteExact: in.Anchor.Quote.Exact,
		Section:    in.Anchor.Section,
		Status:     domain.AnnotationOpen,
		Moderation: moderation,
		Visibility: domain.VisibilityVisible,
		CreatedBy:  p.User.ID,
	}
	return s.annotations.CreateAnnotation(ctx, a, content)
}

func (s *AnnotationService) Update(ctx context.Context, p domain.Principal, siteID, id int64, in UpdateAnnotationInput) (domain.Annotation, error) {
	a, err := s.annotations.AnnotationByID(ctx, siteID, id)
	if err != nil {
		return domain.Annotation{}, err
	}
	if !canManage(p, a.CreatedBy) {
		return domain.Annotation{}, domain.ErrForbidden
	}
	if in.Status == nil && in.Anchor == nil {
		return domain.Annotation{}, domain.Invalid("validation_error", "至少需要 status 或 anchor")
	}

	var anchor *domain.Anchor
	if in.Anchor != nil {
		normalized := *in.Anchor
		normalized.Quote.Exact = strings.TrimSpace(normalized.Quote.Exact)
		if normalized.Quote.Type == "" {
			normalized.Quote.Type = "TextQuoteSelector"
		}
		if err := validateAnchor(normalized); err != nil {
			return domain.Annotation{}, err
		}
		anchor = &normalized
	}
	if in.Status != nil && !in.Status.Valid() {
		return domain.Annotation{}, domain.Invalid("validation_error", "不支持的批注状态")
	}

	if in.Status != nil {
		if err := s.annotations.UpdateAnnotationStatus(ctx, siteID, id, *in.Status); err != nil {
			return domain.Annotation{}, err
		}
	}
	if anchor != nil {
		if err := s.annotations.UpdateAnnotationAnchor(ctx, siteID, id, *anchor); err != nil {
			return domain.Annotation{}, err
		}
	}
	return s.annotations.AnnotationByID(ctx, siteID, id)
}

func (s *AnnotationService) Resolve(ctx context.Context, p domain.Principal, siteID, id int64, resolved bool) error {
	status := domain.AnnotationOpen
	if resolved {
		status = domain.AnnotationResolved
	}
	_, err := s.Update(ctx, p, siteID, id, UpdateAnnotationInput{Status: &status})
	return err
}

func (s *AnnotationService) Delete(ctx context.Context, p domain.Principal, siteID, id int64) error {
	a, err := s.annotations.AnnotationByID(ctx, siteID, id)
	if err != nil {
		return err
	}
	if !canManage(p, a.CreatedBy) {
		return domain.ErrForbidden
	}
	return s.annotations.SoftDeleteAnnotation(ctx, siteID, id)
}

func (s *AnnotationService) AddComment(ctx context.Context, p domain.Principal, site domain.Site, annotationID int64, content string) (domain.Comment, error) {
	if p.User.ID == 0 {
		return domain.Comment{}, domain.ErrUnauthorized
	}
	a, err := s.annotations.AnnotationByID(ctx, site.ID, annotationID)
	if err != nil {
		return domain.Comment{}, err
	}
	if !canManage(p, a.CreatedBy) && (a.Moderation != domain.ModerationApproved || a.Visibility != domain.VisibilityVisible) {
		return domain.Comment{}, domain.ErrNotFound
	}
	content = strings.TrimSpace(content)
	if !withinRunes(content, 1, 4000) {
		return domain.Comment{}, domain.Invalid("validation_error", "回复内容需为 1–4000 字")
	}
	moderation := domain.ModerationApproved
	if site.ModerationMode == domain.ModerationPre {
		moderation = domain.ModerationPending
	}
	return s.comments.CreateComment(ctx, domain.Comment{
		SiteID:       site.ID,
		AnnotationID: annotationID,
		UserID:       p.User.ID,
		Content:      content,
		Moderation:   moderation,
		Visibility:   domain.VisibilityVisible,
	})
}

func (s *AnnotationService) DeleteComment(ctx context.Context, p domain.Principal, siteID, id int64) error {
	c, err := s.comments.CommentByID(ctx, siteID, id)
	if err != nil {
		return err
	}
	if !canManage(p, c.UserID) {
		return domain.ErrForbidden
	}
	return s.comments.SoftDeleteComment(ctx, siteID, id)
}

func canManage(p domain.Principal, authorID int64) bool {
	return p.User.ID == authorID || p.IsSuperadmin || p.IsSiteAdmin
}

func validateAnchor(a domain.Anchor) error {
	a.Quote.Exact = strings.TrimSpace(a.Quote.Exact)
	if !withinRunes(a.Quote.Exact, 1, 2000) {
		return domain.Invalid("validation_error", "锚点文本需为 1–2000 字")
	}
	if !withinRunes(a.Quote.Prefix, 0, 500) || !withinRunes(a.Quote.Suffix, 0, 500) {
		return domain.Invalid("validation_error", "锚点上下文过长")
	}
	if !withinRunes(a.Section, 0, 200) {
		return domain.Invalid("validation_error", "章节标题过长")
	}
	if a.Position != nil && (a.Position.Start < 0 || a.Position.End < a.Position.Start) {
		return domain.Invalid("validation_error", "字符偏移不合法")
	}
	return nil
}

func withinRunes(s string, min, max int) bool {
	n := utf8.RuneCountInString(s)
	return n >= min && n <= max
}
