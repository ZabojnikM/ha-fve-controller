package homeassistant

import (
	"math"
	"strconv"
	"time"
)

type PowerSnapshot struct {
	Enabled      bool               `json:"enabled"`
	Connected    bool               `json:"connected"`
	Status       string             `json:"status"`
	FreshSeconds int                `json:"fresh_seconds"`
	Readings     map[string]Reading `json:"readings"`
}

// Inverter power shares the existing read-only HA polling cycle.
func (r *Reader) PowerSnapshot(now time.Time) PowerSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := PowerSnapshot{Enabled: r.config.Enabled, Connected: r.connected && r.config.Enabled, Status: r.status, FreshSeconds: 60, Readings: map[string]Reading{}}
	reading := Reading{Unit: "W", Quality: "missing"}
	transportFresh := out.Connected && !r.receivedAt.IsZero() && !r.receivedAt.After(now) && now.Sub(r.receivedAt) <= 20*time.Second
	if out.Connected && !transportFresh {
		out.Connected = false
		out.Status = "stale"
	}
	if !out.Enabled {
		out.Status = "disabled"
	}
	if s, exists := r.states["inverter"]; exists && out.Enabled {
		at := s.LastReported
		if at.IsZero() {
			at = s.LastUpdated
		}
		if !at.IsZero() {
			reading.SourceAt = &at
		}
		reading.Quality = "invalid"
		v, err := strconv.ParseFloat(s.State, 64)
		if s.Attributes.Unit == "kW" {
			v *= 1000
		}
		if err == nil && !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 100000 && (s.Attributes.Unit == "W" || s.Attributes.Unit == "kW") {
			reading.Quality = "valid"
			reading.Value = &v
			if at.IsZero() || at.After(now) || now.Sub(at) > 60*time.Second {
				reading.Quality = "stale"
			}
		}
		if !transportFresh {
			reading.Quality = "offline"
		}
	}
	if reading.Quality != "valid" {
		reading.Value = nil
	}
	out.Readings["inverter"] = reading
	return out
}
