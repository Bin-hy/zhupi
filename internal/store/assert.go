package store

// 编译期确认 *Store 实现了所有窄接口。
var (
	_ SiteRepo       = (*Store)(nil)
	_ UserRepo       = (*Store)(nil)
	_ IdentityRepo   = (*Store)(nil)
	_ SessionRepo    = (*Store)(nil)
	_ AuthFlowRepo   = (*Store)(nil)
	_ AnnotationRepo = (*Store)(nil)
	_ CommentRepo    = (*Store)(nil)
	_ ProviderRepo   = (*Store)(nil)
	_ AdminRepo      = (*Store)(nil)
)
