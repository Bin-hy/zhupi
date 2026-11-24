package httpapi

import (
	"context"
	"net/http"

	"github.com/Bin-hy/zhupi/internal/domain"
)

func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

func (s *Server) siteResolve(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		site, err := s.sites.Resolve(r.Context(), r.Header.Get("X-Site-Key"), r.Header.Get("Origin"))
		if err != nil {
			s.writeErr(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), siteCtx, site)))
	})
}

func (s *Server) optionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := bearerToken(r)
		if raw == "" {
			next.ServeHTTP(w, r)
			return
		}
		p, sess, err := s.auth.AuthenticateSession(r.Context(), raw)
		if err != nil {
			s.writeErr(w, r, err)
			return
		}
		ctx := context.WithValue(r.Context(), principalCtx, p)
		ctx = context.WithValue(ctx, sessionCtx, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := PrincipalFrom(r.Context()); ok {
			next.ServeHTTP(w, r)
			return
		}
		raw := bearerToken(r)
		if raw == "" {
			s.writeErr(w, r, domain.ErrUnauthorized)
			return
		}
		p, sess, err := s.auth.AuthenticateSession(r.Context(), raw)
		if err != nil {
			s.writeErr(w, r, err)
			return
		}
		ctx := context.WithValue(r.Context(), principalCtx, p)
		ctx = context.WithValue(ctx, sessionCtx, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) resolveSiteRole(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		site, hasSite := SiteFrom(r.Context())
		p, hasPrincipal := PrincipalFrom(r.Context())
		if !hasSite || !hasPrincipal {
			next.ServeHTTP(w, r)
			return
		}
		p.SiteID = site.ID
		if p.IsSuperadmin {
			p.IsSiteAdmin = true
		} else if isAdmin, err := s.store.IsSiteAdmin(r.Context(), site.ID, p.User.ID); err == nil {
			p.IsSiteAdmin = isAdmin
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalCtx, p)))
	})
}

func (s *Server) requireSiteNotBanned(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		site, hasSite := SiteFrom(r.Context())
		p, hasPrincipal := PrincipalFrom(r.Context())
		if hasSite && hasPrincipal && p.User.ID != 0 {
			banned, err := s.store.IsBanned(r.Context(), site.ID, p.User.ID)
			if err != nil {
				s.writeErr(w, r, err)
				return
			}
			if banned {
				s.writeErr(w, r, domain.Forbidden("site_banned", "你已被禁止在该站点操作"))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireSuperadmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := PrincipalFrom(r.Context())
		if !ok || !p.IsSuperadmin {
			s.writeErr(w, r, domain.ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) readLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.readLimiter.Allow("ip:" + s.clientIP(r)) {
			s.writeErr(w, r, domain.ErrRateLimited)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) writeLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := s.clientIP(r)
		if !s.writeLimiter.Allow("ip:" + ip) {
			s.writeErr(w, r, domain.ErrRateLimited)
			return
		}
		if p, ok := PrincipalFrom(r.Context()); ok && p.User.ID != 0 {
			if !s.userWriteLimiter.Allow("user:" + itoa(p.User.ID)) {
				s.writeErr(w, r, domain.ErrRateLimited)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) publicSite(h http.HandlerFunc) http.Handler {
	return chain(h, s.siteResolve, s.optionalAuth, s.resolveSiteRole, s.readLimit)
}

func (s *Server) writeSite(h http.HandlerFunc) http.Handler {
	return chain(h, s.siteResolve, s.requireAuth, s.resolveSiteRole, s.requireSiteNotBanned, s.writeLimit)
}

func (s *Server) account(h http.HandlerFunc) http.Handler {
	return chain(h, s.requireAuth)
}

func (s *Server) principalForSite(r *http.Request, siteID int64) (domain.Principal, error) {
	p, ok := PrincipalFrom(r.Context())
	if !ok {
		return domain.Principal{}, domain.ErrUnauthorized
	}
	p.SiteID = siteID
	if p.IsSuperadmin {
		p.IsSiteAdmin = true
		return p, nil
	}
	isAdmin, err := s.store.IsSiteAdmin(r.Context(), siteID, p.User.ID)
	if err != nil {
		return domain.Principal{}, err
	}
	p.IsSiteAdmin = isAdmin
	return p, nil
}

func (s *Server) requireModerator(r *http.Request, siteID int64) (domain.Principal, error) {
	p, err := s.principalForSite(r, siteID)
	if err != nil {
		return domain.Principal{}, err
	}
	if !p.CanModerateSite(siteID) {
		return domain.Principal{}, domain.ErrForbidden
	}
	return p, nil
}
