package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/Bin-hy/go-campus/annotation-service/internal/domain"
	"github.com/Bin-hy/go-campus/annotation-service/internal/store"
)

type SiteService struct {
	sites              store.SiteRepo
	allowMissingOrigin bool
}

func NewSiteService(sites store.SiteRepo, allowMissingOrigin bool) *SiteService {
	return &SiteService{sites: sites, allowMissingOrigin: allowMissingOrigin}
}

type CreateSiteInput struct {
	Name           string
	DisplayName    string
	Origins        []string
	ModerationMode domain.ModerationMode
	ThemeColor     string
	DefaultLocale  string
}

type UpdateSiteInput struct {
	DisplayName    string
	Origins        []string
	ModerationMode domain.ModerationMode
	ThemeColor     string
	DefaultLocale  string
	Status         domain.SiteStatus
}

func (s *SiteService) Resolve(ctx context.Context, siteKey, origin string) (domain.Site, error) {
	if strings.TrimSpace(siteKey) == "" {
		return domain.Site{}, domain.Invalid("site_key_required", "缺少 X-Site-Key")
	}
	site, err := s.sites.SiteByKey(ctx, strings.TrimSpace(siteKey))
	if err != nil {
		if _, ok := domain.AsAppError(err); ok {
			return domain.Site{}, domain.NotFound("站点不存在")
		}
		return domain.Site{}, err
	}
	if site.Status != domain.SiteActive {
		return domain.Site{}, domain.Forbidden("site_disabled", "站点已停用")
	}
	if strings.TrimSpace(origin) == "" {
		if !s.allowMissingOrigin {
			return domain.Site{}, domain.Forbidden("origin_required", "缺少请求来源")
		}
		return site, nil
	}
	normalized, err := NormalizeOrigin(origin)
	if err != nil {
		return domain.Site{}, domain.Forbidden("origin_not_allowed", "请求来源不合法")
	}
	for _, allowed := range site.Origins {
		if strings.EqualFold(allowed, normalized) {
			return site, nil
		}
	}
	return domain.Site{}, domain.Forbidden("origin_not_allowed", "该来源不允许访问此站点")
}

// ResolveForLogin 用于顶层导航（浏览器不会发送 Origin）：只校验站点存在且启用。
// 回跳地址仍会在 safeReturnTo 中按站点 Origin 白名单校验。
func (s *SiteService) ResolveForLogin(ctx context.Context, siteKey string) (domain.Site, error) {
	if strings.TrimSpace(siteKey) == "" {
		return domain.Site{}, domain.Invalid("site_key_required", "缺少 siteKey")
	}
	site, err := s.sites.SiteByKey(ctx, strings.TrimSpace(siteKey))
	if err != nil {
		if _, ok := domain.AsAppError(err); ok {
			return domain.Site{}, domain.NotFound("站点不存在")
		}
		return domain.Site{}, err
	}
	if site.Status != domain.SiteActive {
		return domain.Site{}, domain.Forbidden("site_disabled", "站点已停用")
	}
	return site, nil
}

func (s *SiteService) Create(ctx context.Context, in CreateSiteInput) (domain.Site, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || len(name) > 64 {
		return domain.Site{}, domain.Invalid("validation_error", "站点名称需为 1–64 字符")
	}
	origins, err := normalizeOrigins(in.Origins)
	if err != nil {
		return domain.Site{}, err
	}
	if len(origins) == 0 {
		return domain.Site{}, domain.Invalid("validation_error", "至少需要配置一个允许来源")
	}
	mode := in.ModerationMode
	if mode == "" {
		mode = domain.ModerationPost
	}
	if !mode.Valid() {
		return domain.Site{}, domain.Invalid("validation_error", "审核模式只能是 post 或 pre")
	}
	key, err := newSiteKey()
	if err != nil {
		return domain.Site{}, err
	}
	display := strings.TrimSpace(in.DisplayName)
	if display == "" {
		display = name
	}
	theme := strings.TrimSpace(in.ThemeColor)
	if theme == "" {
		theme = "#087f5b"
	}
	locale := strings.TrimSpace(in.DefaultLocale)
	if locale == "" {
		locale = "zh-CN"
	}
	site := domain.Site{
		Key:            key,
		Name:           name,
		DisplayName:    display,
		Status:         domain.SiteActive,
		ModerationMode: mode,
		ThemeColor:     theme,
		DefaultLocale:  locale,
		Origins:        origins,
	}
	return s.sites.CreateSite(ctx, site)
}

func (s *SiteService) Update(ctx context.Context, siteID int64, in UpdateSiteInput) (domain.Site, error) {
	site, err := s.sites.SiteByID(ctx, siteID)
	if err != nil {
		return domain.Site{}, err
	}
	if strings.TrimSpace(in.DisplayName) != "" {
		site.DisplayName = strings.TrimSpace(in.DisplayName)
	}
	if in.Origins != nil {
		origins, err := normalizeOrigins(in.Origins)
		if err != nil {
			return domain.Site{}, err
		}
		if len(origins) == 0 {
			return domain.Site{}, domain.Invalid("validation_error", "至少需要配置一个允许来源")
		}
		site.Origins = origins
	}
	if in.ModerationMode != "" {
		if !in.ModerationMode.Valid() {
			return domain.Site{}, domain.Invalid("validation_error", "审核模式只能是 post 或 pre")
		}
		site.ModerationMode = in.ModerationMode
	}
	if strings.TrimSpace(in.ThemeColor) != "" {
		site.ThemeColor = strings.TrimSpace(in.ThemeColor)
	}
	if strings.TrimSpace(in.DefaultLocale) != "" {
		site.DefaultLocale = strings.TrimSpace(in.DefaultLocale)
	}
	if in.Status != "" {
		if !in.Status.Valid() {
			return domain.Site{}, domain.Invalid("validation_error", "站点状态不合法")
		}
		site.Status = in.Status
	}
	if err := s.sites.UpdateSite(ctx, site); err != nil {
		return domain.Site{}, err
	}
	return s.sites.SiteByID(ctx, siteID)
}

func (s *SiteService) RotateKey(ctx context.Context, siteID int64) (string, error) {
	if _, err := s.sites.SiteByID(ctx, siteID); err != nil {
		return "", err
	}
	key, err := newSiteKey()
	if err != nil {
		return "", err
	}
	if err := s.sites.RotateSiteKey(ctx, siteID, key); err != nil {
		return "", err
	}
	return key, nil
}

func (s *SiteService) List(ctx context.Context, cursor int64, limit int) ([]domain.Site, int64, error) {
	return s.sites.ListSites(ctx, cursor, limit)
}

func newSiteKey() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func normalizeOrigins(raw []string) ([]string, error) {
	seen := map[string]bool{}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if strings.TrimSpace(item) == "" {
			continue
		}
		origin, err := NormalizeOrigin(item)
		if err != nil {
			return nil, domain.Invalid("validation_error", fmt.Sprintf("来源不合法：%s", item))
		}
		if !seen[origin] {
			seen[origin] = true
			out = append(out, origin)
		}
	}
	return out, nil
}

// NormalizeOrigin 只保留 scheme://host[:port]，并统一小写。
func NormalizeOrigin(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid origin")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("invalid scheme")
	}
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host), nil
}
