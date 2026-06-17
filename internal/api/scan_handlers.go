package api

import (
	"net/http"

	"github.com/P0m32Kun/Anchor/internal/scanconfig"
)

func (s *Server) handleGetScanDefaults(w http.ResponseWriter, r *http.Request) {
	cfg := scanconfig.Get()
	if cfg == nil {
		writeJSON(w, http.StatusOK, (&scanconfig.Config{}).DefaultsAPI())
		return
	}
	writeJSON(w, http.StatusOK, cfg.DefaultsAPI())
}
