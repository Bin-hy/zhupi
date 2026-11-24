package domain

type SiteStatus string

const (
	SiteActive   SiteStatus = "active"
	SiteDisabled SiteStatus = "disabled"
)

func (s SiteStatus) Valid() bool { return s == SiteActive || s == SiteDisabled }

type ModerationMode string

const (
	ModerationPost ModerationMode = "post" // 先发后审
	ModerationPre  ModerationMode = "pre"  // 先审后发
)

func (m ModerationMode) Valid() bool { return m == ModerationPost || m == ModerationPre }

type UserStatus string

const (
	UserActive UserStatus = "active"
	UserBanned UserStatus = "banned"
)

func (s UserStatus) Valid() bool { return s == UserActive || s == UserBanned }

type ProviderType string

const (
	ProviderGitHub ProviderType = "github"
	ProviderOIDC   ProviderType = "oidc"
	ProviderDev    ProviderType = "dev" // 仅 DEV_LOGIN=1 时使用
)

func (p ProviderType) Valid() bool {
	return p == ProviderGitHub || p == ProviderOIDC || p == ProviderDev
}

type AnnotationStatus string

const (
	AnnotationOpen     AnnotationStatus = "open"
	AnnotationResolved AnnotationStatus = "resolved"
)

func (s AnnotationStatus) Valid() bool { return s == AnnotationOpen || s == AnnotationResolved }

type ModerationStatus string

const (
	ModerationPending  ModerationStatus = "pending"
	ModerationApproved ModerationStatus = "approved"
	ModerationRejected ModerationStatus = "rejected"
)

func (s ModerationStatus) Valid() bool {
	return s == ModerationPending || s == ModerationApproved || s == ModerationRejected
}

type Visibility string

const (
	VisibilityVisible Visibility = "visible"
	VisibilityHidden  Visibility = "hidden"
	VisibilityDeleted Visibility = "deleted"
)

func (v Visibility) Valid() bool {
	return v == VisibilityVisible || v == VisibilityHidden || v == VisibilityDeleted
}

// FlowMode 区分「登录」与「绑定已有账号」。
type FlowMode string

const (
	FlowLogin FlowMode = "login"
	FlowBind  FlowMode = "bind"
)

func (m FlowMode) Valid() bool { return m == FlowLogin || m == FlowBind }

// ListFilter 是批注列表筛选。
type ListFilter string

const (
	ListAll      ListFilter = "all"
	ListMine     ListFilter = "mine"
	ListResolved ListFilter = "resolved"
	ListPending  ListFilter = "pending"
)

func (f ListFilter) Valid() bool {
	return f == ListAll || f == ListMine || f == ListResolved || f == ListPending
}
