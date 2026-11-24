package httpapi

import (
	"net/http"

	"github.com/Bin-hy/zhupi/internal/domain"
)

func (s *Server) handleAdminListBans(w http.ResponseWriter, r *http.Request) {
	siteID, ok := parseID(r.PathValue("siteID"))
	if !ok {
		s.writeErr(w, r, domain.Invalid("validation_error", "非法的站点 ID"))
		return
	}
	if _, err := s.requireModerator(r, siteID); err != nil {
		s.writeErr(w, r, err)
		return
	}
	list, err := s.store.ListSiteBans(r.Context(), siteID)
	if err != nil {
		s.writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"bans": list})
}
