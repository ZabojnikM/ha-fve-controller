package homeassistant

import (
	"math"
	"strconv"
	"strings"
	"time"
)

type TeslaSnapshot struct {
	Enabled      bool               `json:"enabled"`
	Connected    bool               `json:"connected"`
	Status       string             `json:"status"`
	ReceivedAt   *time.Time         `json:"received_at"`
	FreshSeconds int                `json:"fresh_seconds"`
	Readings     map[string]Reading `json:"readings"`
}

// TeslaSnapshot exposes only selected telemetry, never locations or vehicle identifiers.
func (r *Reader) TeslaSnapshot(now time.Time) TeslaSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	enabled := r.config.Enabled && r.config.TeslaEnabled
	out := TeslaSnapshot{Enabled: enabled, Connected: r.connected && enabled, Status: r.status, FreshSeconds: r.config.TeslaFreshSeconds, Readings: map[string]Reading{}}
	if !enabled {
		out.Status = "disabled"
	}
	if !r.receivedAt.IsZero() && enabled {
		at := r.receivedAt
		out.ReceivedAt = &at
	}
	transportFresh := out.Connected && !r.receivedAt.IsZero() && !r.receivedAt.After(now) && now.Sub(r.receivedAt) <= 20*time.Second
	if out.Connected && !transportFresh {
		out.Connected = false
		out.Status = "stale"
	}
	for key, unit := range map[string]string{"soc": "%", "connected": "", "charging": "", "current": "A", "power": "W", "current_limit": "A"} {
		reading := Reading{Unit: unit, Quality: "missing"}
		if r.config.TeslaEntities[key] == "" {
			reading.Quality = "not_configured"
			out.Readings[key] = reading
			continue
		}
		s, exists := r.states["tesla_"+key]
		if exists && enabled {
			at := s.LastReported
			if at.IsZero() {
				at = s.LastUpdated
			}
			if !at.IsZero() {
				reading.SourceAt = &at
			}
			reading.Quality = "invalid"
			switch key {
			case "connected":
				if s.State == "on" || s.State == "off" {
					v := 0.0
					if s.State == "on" {
						v = 1
					}
					reading.Value = &v
					reading.Quality = "valid"
				}
			case "charging":
				// Only recognized charge states cross the API boundary; no raw strings.
				state := strings.ToLower(strings.TrimSpace(s.State))
				if state == "nopower" {
					state = "no_power"
				}
				switch state {
				case "charging", "complete", "disconnected", "stopped", "starting", "no_power":
					reading.Text = &state
					reading.Quality = "valid"
				}
			default:
				v, err := strconv.ParseFloat(s.State, 64)
				if err == nil && !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 {
					valid := false
					switch key {
					case "soc":
						valid = s.Attributes.Unit == "%" && v <= 100
					case "current", "current_limit":
						valid = s.Attributes.Unit == "A" && v <= 100
					case "power":
						if s.Attributes.Unit == "kW" {
							v *= 1000
						}
						valid = (s.Attributes.Unit == "W" || s.Attributes.Unit == "kW") && v <= 100000
					}
					if valid {
						reading.Value = &v
						reading.Quality = "valid"
					}
				}
			}
			if reading.Quality == "valid" && (at.IsZero() || at.After(now) || now.Sub(at) > time.Duration(r.config.TeslaFreshSeconds)*time.Second) {
				reading.Quality = "stale"
			}
			if !transportFresh {
				reading.Quality = "offline"
			}
		}
		if reading.Quality != "valid" {
			reading.Value = nil
			reading.Text = nil
		}
		out.Readings[key] = reading
	}
	return out
}
