package service

import (
	"context"

	"github.com/Bin-hy/go-campus/annotation-service/internal/domain"
	"github.com/Bin-hy/go-campus/annotation-service/internal/store"
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
	Comments         []domain.Comment    `json:"comments"`
	AnnotationCursor int64               `json:"annotationCursor"`
	CommentCursor    int64               `json:"commentCursor"`
}

func (m *ModerationService) Queue(ctx context.Context, p domain.Principal, siteID int64,
	status domain.ModerationStatus, annotationCursor, commentCursor int64, limit int) (Queue, error) {
	if !p.CanModerateSite(siteID) {
		return Queue{}, domain.ErrForbidden
	}
	if status == "" {
		status = domain.ModerationPending
	}
	if !status.Valid() {
		return Queue{}, domain.Invalid("validation_error", "审核状态不合法")
	}
	annotations, nextA, err := m.annotations.ListAnnotationsForModeration(ctx, siteID, status, annotationCursor, limit)
	if err != nil {
		return Queue{}, err
	}
	comments, nextC, err := m.comments.ListCommentsForModeration(ctx, siteID, status, commentCursor, limit)
	if err != nil {
		return Queue{}, err
	}
	return Queue{Annotations: annotations, Comments: comments, AnnotationCursor: nextA, CommentCursor: nextC}, nil
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
