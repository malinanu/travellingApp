package httpapi

import (
	"net/http"
)

type LocationConfig struct {
	Enabled bool    `json:"enabled"`
	Name    string  `json:"name"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Zoom    float64 `json:"zoom"`
}

type PublicConfigResponse struct {
	DefaultLocation *LocationConfig `json:"defaultLocation"`
}

// handleConfig returns public server configurations such as staging location overrides.
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	var loc *LocationConfig
	if s.DefaultLocationMirissa {
		loc = &LocationConfig{
			Enabled: true,
			Name:    "Mirissa",
			Lat:     5.9483,
			Lng:     80.4552,
			Zoom:    14.5,
		}
	}
	writeJSON(w, http.StatusOK, PublicConfigResponse{
		DefaultLocation: loc,
	})
}
