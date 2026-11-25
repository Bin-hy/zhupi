package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	authpkg "github.com/Bin-hy/zhupi/internal/auth"
	"github.com/Bin-hy/zhupi/internal/config"
	"github.com/Bin-hy/zhupi/internal/crypto"
	"github.com/Bin-hy/zhupi/internal/domain"
	"github.com/Bin-hy/zhupi/internal/httpapi"
	"github.com/Bin-hy/zhupi/internal/service"
	"github.com/Bin-hy/zhupi/internal/store"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "serve":
			runServe()
			return
		case "admin":
			runAdmin(os.Args[2:])
			return
		}
	}
	runServe()
}

func logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func runServe() {
	log := logger()
	cfg, err := config.Load()
	if err != nil {
		log.Error("配置错误", "err", err)
		os.Exit(1)
	}
	st, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Error("打开数据库失败", "err", err)
		os.Exit(1)
	}
	defer st.Close()
	if err := st.Migrate(context.Background()); err != nil {
		log.Error("数据库迁移失败", "err", err)
		os.Exit(1)
	}
	box, err := crypto.New(cfg.SecretKey)
	if err != nil {
		log.Error("初始化加密失败", "err", err)
		os.Exit(1)
	}

	github := authpkg.NewGitHubClient(authpkg.GitHubConfig{
		ClientID:     cfg.GitHubClientID,
		ClientSecret: cfg.GitHubSecret,
		RedirectURL:  cfg.PublicBaseURL + "/api/v1/auth/github/callback",
	}, nil)
	oidcClient := authpkg.NewOIDCClient(nil)

	sites := service.NewSiteService(st, cfg.AllowMissingOrigin)
	authSvc := service.NewAuthService(cfg, st, st, st, st, st, st, box, github, oidcClient)
	boot, err := authSvc.BootstrapLocalAdmin(context.Background())
	if err != nil {
		log.Error("初始化本地管理员失败", "err", err)
		os.Exit(1)
	}
	switch {
	case boot.Created:
		log.Info("已创建本地超级管理员", "username", boot.Username, "mustChangePassword", boot.MustChangePassword)
	case boot.Reset:
		log.Warn("已按 LOCAL_ADMIN_RESET_PASSWORD 重置本地管理员密码", "username", boot.Username, "mustChangePassword", boot.MustChangePassword)
	case boot.Skipped:
		log.Info("本地管理员已存在，跳过初始化", "username", boot.Username)
	}
	annotations := service.NewAnnotationService(st, st)
	moderation := service.NewModerationService(st, st, st, st, st)
	admin := service.NewAdminService(st, st, st, st, st, st)
	srv := httpapi.New(cfg, st, sites, authSvc, annotations, moderation, admin, log)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_ = st.PurgeExpiredSessions(ctx)
			_ = st.PurgeExpiredAuthFlows(ctx)
			cancel()
		}
	}()

	go func() {
		log.Info("zhupi 启动", "addr", cfg.Addr, "devLogin", cfg.DevLogin, "github", cfg.GitHubConfigured())
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("服务启动失败", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
	log.Info("已退出")
}

func runAdmin(args []string) {
	log := logger()
	if len(args) == 0 {
		fmt.Println("用法: zhupi admin <migrate|site|provider|promote|users|password> [...]")
		os.Exit(2)
	}
	cfg, err := config.Load()
	if err != nil {
		log.Error("配置错误", "err", err)
		os.Exit(1)
	}
	st, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Error("打开数据库失败", "err", err)
		os.Exit(1)
	}
	defer st.Close()
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		log.Error("数据库迁移失败", "err", err)
		os.Exit(1)
	}

	sites := service.NewSiteService(st, true)
	adminSvc := service.NewAdminService(st, st, st, st, st, st)
	box, err := crypto.New(cfg.SecretKey)
	if err != nil {
		log.Error("初始化加密失败", "err", err)
		os.Exit(1)
	}
	authSvc := service.NewAuthService(cfg, st, st, st, st, st, st, box, nil, authpkg.NewOIDCClient(nil))
	super := domain.Principal{User: domain.User{ID: 0, DisplayName: "cli"}, IsSuperadmin: true}

	switch args[0] {
	case "migrate":
		fmt.Println("迁移完成")
	case "password":
		runPasswordCLI(ctx, authSvc, args[1:])
	case "site":
		runSiteCLI(ctx, st, sites, adminSvc, super, args[1:])
	case "provider":
		runProviderCLI(ctx, st, authSvc, args[1:])
	case "promote":
		fs := flag.NewFlagSet("promote", flag.ExitOnError)
		login := fs.String("github-login", "", "GitHub 登录名")
		userID := fs.Int64("user-id", 0, "用户 ID")
		_ = fs.Parse(args[1:])
		switch {
		case *login != "":
			id, err := adminSvc.PromoteByGitHubHandle(ctx, *login)
			if err != nil {
				fail("提升失败: %v", err)
			}
			fmt.Printf("已将用户 %d 提升为超级管理员\n", id)
		case *userID > 0:
			if err := adminSvc.PromoteUser(ctx, *userID); err != nil {
				fail("提升失败: %v", err)
			}
			fmt.Printf("已将用户 %d 提升为超级管理员\n", *userID)
		default:
			fail("需要 --github-login 或 --user-id")
		}
	case "users":
		if len(args) < 2 || args[1] != "list" {
			fail("用法: admin users list [--json]")
		}
		fs := flag.NewFlagSet("users list", flag.ExitOnError)
		jsonOut := fs.Bool("json", false, "输出 JSON")
		_ = fs.Parse(args[2:])
		list, _, err := adminSvc.ListUsers(ctx, super, 0, 200)
		if err != nil {
			fail("读取用户失败: %v", err)
		}
		if *jsonOut {
			printJSON(list)
			return
		}
		fmt.Printf("%-6s %-24s %-10s %s\n", "ID", "名称", "状态", "超管")
		for _, u := range list {
			fmt.Printf("%-6d %-24s %-10s %v\n", u.ID, u.DisplayName, u.Status, u.IsSuperadmin)
		}
	default:
		fail("未知命令: %s", args[0])
	}
}

func runPasswordCLI(ctx context.Context, authSvc *service.AuthService, args []string) {
	if len(args) == 0 || args[0] != "reset" {
		fail("用法: admin password reset --username <name> [--password <pwd>]")
	}
	fs := flag.NewFlagSet("password reset", flag.ExitOnError)
	username := fs.String("username", "admin", "本地管理员用户名")
	password := fs.String("password", "", "新密码；省略时生成随机密码")
	_ = fs.Parse(args[1:])

	pwd := *password
	generated := false
	if pwd == "" {
		var err error
		pwd, err = authpkg.GeneratePassword()
		if err != nil {
			fail("生成随机密码失败: %v", err)
		}
		generated = true
	}
	cred, err := authSvc.ResetLocalPassword(ctx, *username, pwd)
	if err != nil {
		fail("重置密码失败: %v", err)
	}
	if generated {
		fmt.Printf("已重置 %s 的密码：%s\n", cred.Username, pwd)
	} else {
		fmt.Printf("已重置 %s 的密码\n", cred.Username)
	}
	fmt.Println("该账号的所有会话已失效。")
}

func runSiteCLI(ctx context.Context, st *store.Store, sites *service.SiteService,
	adminSvc *service.AdminService, super domain.Principal, args []string) {
	if len(args) == 0 {
		fail("用法: admin site <create|list|rotate-key|add-admin|remove-admin>")
	}
	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("site create", flag.ExitOnError)
		name := fs.String("name", "", "内部名称（唯一）")
		display := fs.String("display-name", "", "展示名")
		moderation := fs.String("moderation", "post", "post|pre")
		theme := fs.String("theme", "#087f5b", "主题色")
		locale := fs.String("locale", "zh-CN", "默认语言")
		var origins multiFlag
		fs.Var(&origins, "origin", "允许来源，可重复")
		_ = fs.Parse(args[1:])
		site, err := sites.Create(ctx, service.CreateSiteInput{
			Name: *name, DisplayName: *display, Origins: origins,
			ModerationMode: domain.ModerationMode(*moderation), ThemeColor: *theme, DefaultLocale: *locale,
		})
		if err != nil {
			fail("创建站点失败: %v", err)
		}
		printJSON(site)
	case "list":
		list, _, err := sites.List(ctx, 0, 200)
		if err != nil {
			fail("读取站点失败: %v", err)
		}
		fmt.Printf("%-6s %-22s %-16s %-10s %s\n", "ID", "名称", "siteKey", "状态", "来源")
		for _, s := range list {
			fmt.Printf("%-6d %-22s %-16s %-10s %s\n", s.ID, s.Name, s.Key, s.Status, strings.Join(s.Origins, ","))
		}
	case "rotate-key":
		fs := flag.NewFlagSet("site rotate-key", flag.ExitOnError)
		key := fs.String("site", "", "siteKey 或 ID")
		_ = fs.Parse(args[1:])
		site := mustSite(ctx, st, *key)
		newKey, err := sites.RotateKey(ctx, site.ID)
		if err != nil {
			fail("轮换失败: %v", err)
		}
		fmt.Println(newKey)
	case "add-admin", "remove-admin":
		fs := flag.NewFlagSet("site "+args[0], flag.ExitOnError)
		key := fs.String("site", "", "siteKey 或 ID")
		user := fs.String("user", "", "用户 ID 或 GitHub 登录名")
		_ = fs.Parse(args[1:])
		site := mustSite(ctx, st, *key)
		u := mustUser(ctx, st, *user)
		if args[0] == "add-admin" {
			if err := adminSvc.AddSiteAdmin(ctx, super, site.ID, u.ID); err != nil {
				fail("添加失败: %v", err)
			}
			fmt.Println("已添加站点管理员")
		} else {
			if err := adminSvc.RemoveSiteAdmin(ctx, super, site.ID, u.ID); err != nil {
				fail("移除失败: %v", err)
			}
			fmt.Println("已移除站点管理员")
		}
	default:
		fail("未知 site 子命令: %s", args[0])
	}
}

// providerCLIView 在提供商字段之外附带站点名，便于 CLI 输出直接定位。
type providerCLIView struct {
	domain.OIDCProvider
	SiteName string `json:"siteName"`
}

func listAllSites(ctx context.Context, st *store.Store) ([]domain.Site, error) {
	var out []domain.Site
	cursor := int64(0)
	for {
		sites, next, err := st.ListSites(ctx, cursor, 200)
		if err != nil {
			return nil, err
		}
		out = append(out, sites...)
		if next == 0 {
			return out, nil
		}
		cursor = next
	}
}

func runProviderCLI(ctx context.Context, st *store.Store, authSvc *service.AuthService, args []string) {
	if len(args) == 0 {
		fail("用法: admin provider <list|add|update|delete> [...]")
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("provider list", flag.ExitOnError)
		siteRef := fs.String("site", "", "siteKey 或 ID；省略时列出全部站点")
		_ = fs.Parse(args[1:])
		out := make([]providerCLIView, 0)
		if *siteRef != "" {
			site := mustSite(ctx, st, *siteRef)
			list, err := st.ListProvidersBySite(ctx, site.ID, false)
			if err != nil {
				fail("读取提供商失败: %v", err)
			}
			for _, p := range list {
				out = append(out, providerCLIView{OIDCProvider: p, SiteName: site.Name})
			}
		} else {
			sites, err := listAllSites(ctx, st)
			if err != nil {
				fail("读取站点失败: %v", err)
			}
			for _, site := range sites {
				list, err := st.ListProvidersBySite(ctx, site.ID, false)
				if err != nil {
					fail("读取站点 %d 的提供商失败: %v", site.ID, err)
				}
				for _, p := range list {
					out = append(out, providerCLIView{OIDCProvider: p, SiteName: site.Name})
				}
			}
		}
		printJSON(out)
	case "add":
		fs := flag.NewFlagSet("provider add", flag.ExitOnError)
		siteRef := fs.String("site", "", "siteKey 或 ID")
		name := fs.String("name", "", "展示名，例如 yufu")
		issuer := fs.String("issuer", "", "OIDC issuer，例如 https://cobotrix.gataccay.com")
		clientID := fs.String("client-id", "", "OIDC client id")
		clientSecret := fs.String("client-secret", "", "OIDC client secret")
		clientSecretFile := fs.String("client-secret-file", "", "从文件读取 client secret（优先于 --client-secret）")
		scopesRaw := fs.String("scopes", "openid,profile,email", "逗号分隔的 scopes")
		domainsRaw := fs.String("allowed-email-domains", "", "逗号分隔的允许邮箱域")
		enabledRaw := fs.String("enabled", "true", "true|false")
		_ = fs.Parse(args[1:])
		secret, err := readSecret(*clientSecret, *clientSecretFile)
		if err != nil {
			fail("读取 client secret 失败: %v", err)
		}
		providerName := strings.TrimSpace(*name)
		if len(providerName) == 0 || len(providerName) > 64 {
			fail("提供商名称需为 1–64 字符")
		}
		enabled, ok := parseProviderEnabled(*enabledRaw)
		if !ok {
			fail("--enabled 只能是 true 或 false")
		}
		if *siteRef == "" || strings.TrimSpace(*issuer) == "" ||
			strings.TrimSpace(*clientID) == "" || secret == "" {
			fail("需要 --site、--name、--issuer、--client-id、--client-secret 或 --client-secret-file")
		}
		site := mustSite(ctx, st, *siteRef)
		issuerURL := strings.TrimRight(strings.TrimSpace(*issuer), "/")
		if err := authSvc.ValidateIssuer(ctx, issuerURL); err != nil {
			fail("OIDC Discovery 校验失败: %v", err)
		}
		cipher, err := authSvc.EncryptSecret(secret)
		if err != nil {
			fail("加密 client secret 失败: %v", err)
		}
		p, err := st.CreateProvider(ctx, domain.OIDCProvider{
			SiteID: site.ID, Name: providerName, Issuer: issuerURL,
			ClientID: strings.TrimSpace(*clientID), ClientSecretCipher: cipher,
			Scopes: providerScopes(*scopesRaw), AllowedEmailDomains: normalizeProviderDomains(*domainsRaw), Enabled: enabled,
		})
		if err != nil {
			fail("创建提供商失败: %v", err)
		}
		printJSON(providerCLIView{OIDCProvider: p, SiteName: site.Name})
	case "update":
		fs := flag.NewFlagSet("provider update", flag.ExitOnError)
		id := fs.Int64("id", 0, "提供商 ID")
		name := fs.String("name", "", "展示名")
		issuer := fs.String("issuer", "", "OIDC issuer")
		clientID := fs.String("client-id", "", "OIDC client id")
		clientSecret := fs.String("client-secret", "", "OIDC client secret；留空保持原值")
		clientSecretFile := fs.String("client-secret-file", "", "从文件读取 client secret；留空保持原值")
		scopesRaw := fs.String("scopes", "", "逗号分隔的 scopes；留空保持原值")
		domainsRaw := fs.String("allowed-email-domains", "", "逗号分隔的允许邮箱域；留空保持原值")
		enabledRaw := fs.String("enabled", "", "true|false；留空保持原值")
		_ = fs.Parse(args[1:])
		if *id <= 0 {
			fail("需要 --id")
		}
		p, err := st.ProviderByIDAny(ctx, *id)
		if err != nil {
			fail("提供商不存在: %v", err)
		}
		changed := false
		issuerChanged := false
		if providerName := strings.TrimSpace(*name); providerName != "" {
			if len(providerName) > 64 {
				fail("提供商名称需为 1–64 字符")
			}
			if providerName != p.Name {
				p.Name = providerName
				changed = true
			}
		}
		if raw := strings.TrimSpace(*issuer); raw != "" {
			issuer := strings.TrimRight(raw, "/")
			if issuer != p.Issuer {
				p.Issuer = issuer
				issuerChanged = true
				changed = true
			}
		}
		if clientID := strings.TrimSpace(*clientID); clientID != "" && clientID != p.ClientID {
			p.ClientID = clientID
			changed = true
		}
		secret, err := readSecret(*clientSecret, *clientSecretFile)
		if err != nil {
			fail("读取 client secret 失败: %v", err)
		}
		if secret != "" {
			cipher, err := authSvc.EncryptSecret(secret)
			if err != nil {
				fail("加密 client secret 失败: %v", err)
			}
			p.ClientSecretCipher = cipher
			changed = true
		}
		if raw := strings.TrimSpace(*scopesRaw); raw != "" {
			p.Scopes = providerScopes(raw)
			changed = true
		}
		if raw := strings.TrimSpace(*domainsRaw); raw != "" {
			p.AllowedEmailDomains = normalizeProviderDomains(raw)
			changed = true
		}
		if raw := strings.TrimSpace(*enabledRaw); raw != "" {
			enabled, ok := parseProviderEnabled(raw)
			if !ok {
				fail("--enabled 只能是 true 或 false")
			}
			if enabled != p.Enabled {
				p.Enabled = enabled
				changed = true
			}
		}
		if !changed {
			fail("没有需要更新的字段")
		}
		if issuerChanged {
			if err := authSvc.ValidateIssuer(ctx, p.Issuer); err != nil {
				fail("OIDC Discovery 校验失败: %v", err)
			}
		}
		if err := st.UpdateProvider(ctx, p); err != nil {
			fail("更新提供商失败: %v", err)
		}
		site, err := st.SiteByID(ctx, p.SiteID)
		if err != nil {
			fail("读取站点失败: %v", err)
		}
		printJSON(providerCLIView{OIDCProvider: p, SiteName: site.Name})
	case "delete":
		fs := flag.NewFlagSet("provider delete", flag.ExitOnError)
		id := fs.Int64("id", 0, "提供商 ID")
		_ = fs.Parse(args[1:])
		if *id <= 0 {
			fail("需要 --id")
		}
		p, err := st.ProviderByIDAny(ctx, *id)
		if err != nil {
			fail("提供商不存在: %v", err)
		}
		if err := st.DeleteProvider(ctx, p.SiteID, p.ID); err != nil {
			fail("删除提供商失败: %v", err)
		}
		fmt.Printf("已删除提供商 %d (%s)\n", p.ID, p.Name)
	default:
		fail("未知 provider 子命令: %s", args[0])
	}
}

func splitList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func providerScopes(raw string) []string {
	scopes := uniqueStrings(splitList(raw))
	if len(scopes) == 0 {
		return []string{"openid", "email", "profile"}
	}
	return scopes
}

func normalizeProviderDomains(raw string) []string {
	domains := splitList(raw)
	out := make([]string, 0, len(domains))
	for _, d := range domains {
		d = strings.ToLower(strings.TrimPrefix(d, "@"))
		if d != "" {
			out = append(out, d)
		}
	}
	return uniqueStrings(out)
}

func uniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func parseProviderEnabled(raw string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes":
		return true, true
	case "false", "0", "no":
		return false, true
	default:
		return false, false
	}
}

func readSecret(raw, file string) (string, error) {
	if strings.TrimSpace(file) != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	return strings.TrimSpace(raw), nil
}

type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

func mustSite(ctx context.Context, st *store.Store, raw string) domain.Site {
	if raw == "" {
		fail("需要 --site")
	}
	if id, err := strconv.ParseInt(raw, 10, 64); err == nil && id > 0 {
		site, err := st.SiteByID(ctx, id)
		if err != nil {
			fail("站点不存在: %v", err)
		}
		return site
	}
	site, err := st.SiteByKey(ctx, raw)
	if err != nil {
		fail("站点不存在: %v", err)
	}
	return site
}

func mustUser(ctx context.Context, st *store.Store, raw string) domain.User {
	if raw == "" {
		fail("需要 --user")
	}
	if id, err := strconv.ParseInt(raw, 10, 64); err == nil && id > 0 {
		u, err := st.UserByID(ctx, id)
		if err != nil {
			fail("用户不存在: %v", err)
		}
		return u
	}
	identity, err := st.IdentityByHandle(ctx, domain.ProviderGitHub, raw)
	if err != nil {
		fail("GitHub 用户不存在（请先登录一次）: %v", err)
	}
	u, err := st.UserByID(ctx, identity.UserID)
	if err != nil {
		fail("用户不存在: %v", err)
	}
	return u
}

func printJSON(v any) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
