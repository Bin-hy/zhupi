package service

import (
	"context"
	"strings"

	"github.com/Bin-hy/zhupi/internal/domain"
	"github.com/Bin-hy/zhupi/internal/store"
)

type ModerationAction string

const (
	ActionApprove ModerationAction = "approve"
	ActionReject  ModerationAction = "reject"
	ActionHide    ModerationAction = "hide"
	ActionRestore ModerationAction = "restore"
)

func (a ModerationAction) Valid() bool {
	switch a {
	case ActionApprove, ActionReject, ActionHide, ActionRestore:
		return true
	}
	return false
}

type ModerationService struct {
	annotations store.AnnotationRepo
	comments    store.CommentRepo
	admins      store.AdminRepo
	users       store.UserRepo
	sessions    store.SessionRepo
}

func NewModerationService(annotations store.AnnotationRepo, comments store.CommentRepo,
	admins store.AdminRepo, users store.UserRepo, sessions store.SessionRepo) *ModerationService {
	return &ModerationService{annotations: annotations, comments: comments, admins: admins, users: users, sessions: sessions}
}

type Queue struct {
	Annotations      []domain.Annotation `json:"annotations"`
	Comments         []ModerationComment `json:"comments"`
	AnnotationCursor int64               `json:"annotationCursor"`
	CommentCursor    int64               `json:"commentCursor"`
	Counts           ModerationCounts    `json:"counts"`
}

// AnnotationBrief 是评论审核时展示的父批注摘要，避免审核员离开当前列表。
type AnnotationBrief struct {
	ID         int64                   `json:"id"`
	PageKey    string                  `json:"pageKey"`
	PageTitle  string                  `json:"pageTitle"`
	QuoteExact string                  `json:"quoteExact"`
	Body       string                  `json:"body,omitempty"`
	Section    string                  `json:"section,omitempty"`
	Status     domain.AnnotationStatus `json:"status"`
	Moderation domain.ModerationStatus `json:"moderation"`
	Visibility domain.Visibility       `json:"visibility"`
	Author     domain.Author           `json:"author"`
}

// ModerationComment 在评论字段之外附带父批注，前端可直接展示引用与原文入口。
type ModerationComment struct {
	domain.Comment
	Parent *AnnotationBrief `json:"parent,omitempty"`
}

type StatusCounts struct {
	Pending  int `json:"pending"`
	Approved int `json:"approved"`
	Rejected int `json:"rejected"`
}

type ModerationCounts struct {
	Annotations StatusCounts `json:"annotations"`
	Comments    StatusCounts `json:"comments"`
}

func (m *ModerationService) Queue(ctx context.Context, p domain.Principal, siteID int64,
	status domain.ModerationStatus, keyword, kind string, annotationCursor, commentCursor int64, limit int) (Queue, error) {
	if !p.CanModerateSite(siteID) {
		return Queue{}, domain.ErrForbidden
	}
	if status == "" {
		status = domain.ModerationPending
	}
	if !status.Valid() {
		return Queue{}, domain.Invalid("validation_error", "审核状态不合法")
	}
	keyword = strings.TrimSpace(keyword)
	if len([]rune(keyword)) > 100 {
		keyword = string([]rune(keyword)[:100])
	}
	if kind == "" {
		kind = "all"
	}
	if kind != "all" && kind != "annotation" && kind != "comment" {
		return Queue{}, domain.Invalid("validation_error", "不支持的审核类型")
	}
	annotations := []domain.Annotation{}
	comments := []domain.Comment{}
	var nextA, nextC int64
	var err error
	if kind != "comment" {
		annotations, nextA, err = m.annotations.ListAnnotationsForModeration(ctx, siteID, status, keyword, annotationCursor, limit)
		if err != nil {
			return Queue{}, err
		}
	}
	if kind != "annotation" {
		comments, nextC, err = m.comments.ListCommentsForModeration(ctx, siteID, status, keyword, commentCursor, limit)
		if err != nil {
			return Queue{}, err
		}
	}
	parents := map[int64]domain.Annotation{}
	bodies := map[int64]string{}
	if len(comments) > 0 {
		ids := make([]int64, 0, len(comments))
		seen := map[int64]bool{}
		for _, c := range comments {
			if !seen[c.AnnotationID] {
				seen[c.AnnotationID] = true
				ids = append(ids, c.AnnotationID)
			}
		}
		parents, err = m.annotations.AnnotationsByIDs(ctx, siteID, ids)
		if err != nil {
			return Queue{}, err
		}
		bodies, err = m.annotations.FirstCommentsByAnnotationIDs(ctx, siteID, ids)
		if err != nil {
			return Queue{}, err
		}
	}
	out := make([]ModerationComment, len(comments))
	for i, c := range comments {
		out[i] = ModerationComment{Comment: c}
		if a, ok := parents[c.AnnotationID]; ok {
			out[i].Parent = &AnnotationBrief{
				ID: a.ID, PageKey: a.PageKey, PageTitle: a.PageTitle, QuoteExact: a.QuoteExact,
				Body:    bodies[a.ID],
				Section: a.Section, Status: a.Status, Moderation: a.Moderation,
				Visibility: a.Visibility, Author: a.Author,
			}
		}
	}
	annotationCounts, err := m.annotations.CountAnnotationsForModeration(ctx, siteID)
	if err != nil {
		return Queue{}, err
	}
	commentCounts, err := m.comments.CountCommentsForModeration(ctx, siteID)
	if err != nil {
		return Queue{}, err
	}
	return Queue{
		Annotations:      annotations,
		Comments:         out,
		AnnotationCursor: nextA,
		CommentCursor:    nextC,
		Counts: ModerationCounts{
			Annotations: StatusCounts{
				Pending: annotationCounts[domain.ModerationPending], Approved: annotationCounts[domain.ModerationApproved],
				Rejected: annotationCounts[domain.ModerationRejected],
			},
			Comments: StatusCounts{
				Pending: commentCounts[domain.ModerationPending], Approved: commentCounts[domain.ModerationApproved],
				Rejected: commentCounts[domain.ModerationRejected],
			},
		},
	}, nil
}

func (m *ModerationService) ModerateAnnotation(ctx context.Context, p domain.Principal, siteID, id int64, action ModerationAction) error {
	if !p.CanModerateSite(siteID) {
		return domain.ErrForbidden
	}
	if !action.Valid() {
		return domain.Invalid("validation_error", "审核动作不合法")
	}
	if _, err := m.annotations.AnnotationByID(ctx, siteID, id); err != nil {
		return err
	}
	switch action {
	case ActionApprove:
		if err := m.annotations.SetAnnotationModeration(ctx, siteID, id, domain.ModerationApproved); err != nil {
			return err
		}
		return m.annotations.SetAnnotationVisibility(ctx, siteID, id, domain.VisibilityVisible)
	case ActionReject:
		return m.annotations.SetAnnotationModeration(ctx, siteID, id, domain.ModerationRejected)
	case ActionHide:
		return m.annotations.SetAnnotationVisibility(ctx, siteID, id, domain.VisibilityHidden)
	case ActionRestore:
		return m.annotations.SetAnnotationVisibility(ctx, siteID, id, domain.VisibilityVisible)
	}
	return nil
}

// AnnotationDetail 返回批注及其完整讨论串，供审核员在列表内查看上下文。
func (m *ModerationService) AnnotationDetail(ctx context.Context, p domain.Principal, siteID, id int64) (domain.Annotation, error) {
	if !p.CanModerateSite(siteID) {
		return domain.Annotation{}, domain.ErrForbidden
	}
	a, err := m.annotations.AnnotationByID(ctx, siteID, id)
	if err != nil {
		return domain.Annotation{}, err
	}
	comments, err := m.comments.ListCommentsByAnnotation(ctx, siteID, id, p.User.ID, true)
	if err != nil {
		return domain.Annotation{}, err
	}
	a.Comments = comments
	return a, nil
}

func (m *ModerationService) ModerateComment(ctx context.Context, p domain.Principal, siteID, id int64, action ModerationAction) error {
	if !p.CanModerateSite(siteID) {
		return domain.ErrForbidden
	}
	if !action.Valid() {
		return domain.Invalid("validation_error", "审核动作不合法")
	}
	if _, err := m.comments.CommentByID(ctx, siteID, id); err != nil {
		return err
	}
	switch action {
	case ActionApprove:
		return m.comments.SetCommentModeration(ctx, siteID, id, domain.ModerationApproved)
	case ActionReject:
		return m.comments.SetCommentModeration(ctx, siteID, id, domain.ModerationRejected)
	case ActionHide, ActionRestore:
		// 评论的显示/隐藏通过审核状态表达：hide 用 rejected，restore 用 approved。
		if action == ActionHide {
			return m.comments.SetCommentModeration(ctx, siteID, id, domain.ModerationRejected)
		}
		return m.comments.SetCommentModeration(ctx, siteID, id, domain.ModerationApproved)
	}
	return nil
}

func (m *ModerationService) BanUser(ctx context.Context, p domain.Principal, siteID, userID int64, reason string) error {
	if !p.CanModerateSite(siteID) {
		return domain.ErrForbidden
	}
	if _, err := m.users.UserByID(ctx, userID); err != nil {
		return err
	}
	return m.admins.BanSite(ctx, domain.SiteBan{
		SiteID: siteID, UserID: userID, BannedBy: p.User.ID, Reason: truncate(reason, 200),
	})
}

func (m *ModerationService) UnbanUser(ctx context.Context, p domain.Principal, siteID, userID int64) error {
	if !p.CanModerateSite(siteID) {
		return domain.ErrForbidden
	}
	return m.admins.UnbanSite(ctx, siteID, userID)
}
