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

	authpkg "github.com/Bin-hy/go-campus/annotation-service/internal/auth"
	"github.com/Bin-hy/go-campus/annotation-service/internal/config"
	"github.com/Bin-hy/go-campus/annotation-service/internal/crypto"
	"github.com/Bin-hy/go-campus/annotation-service/internal/domain"
	"github.com/Bin-hy/go-campus/annotation-service/internal/httpapi"
	"github.com/Bin-hy/go-campus/annotation-service/internal/service"
	"github.com/Bin-hy/go-campus/annotation-service/internal/store"
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
	authSvc := service.NewAuthService(cfg, st, st, st, st, st, box, github, oidcClient)
	annotations := service.NewAnnotationService(st, st)
	moderation := service.NewModerationService(st, st, st, st, st)
	admin := service.NewAdminService(st, st, st, st, st)
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
		log.Info("annotation-service 启动", "addr", cfg.Addr, "devLogin", cfg.DevLogin, "github", cfg.GitHubConfigured())
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
		fmt.Println("用法: annotation-service admin <migrate|site|promote|users> [...]")
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
	adminSvc := service.NewAdminService(st, st, st, st, st)
	super := domain.Principal{User: domain.User{ID: 0, DisplayName: "cli"}, IsSuperadmin: true}

	switch args[0] {
	case "migrate":
		fmt.Println("迁移完成")
	case "site":
		runSiteCLI(ctx, st, sites, adminSvc, super, args[1:])
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
