// Package homeassistant reads HA state only; it never calls services or writes states.
package homeassistant

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Enabled                 bool              `json:"ha_enabled"`
	TemperatureFreshSeconds int               `json:"ha_temperature_fresh_seconds"`
	Entities                map[string]string `json:"ha_entities"`
	TeslaEnabled            bool              `json:"tesla_enabled"`
	TeslaFreshSeconds       int               `json:"tesla_fresh_seconds"`
	TeslaEntities           map[string]string `json:"tesla_entities"`
}

func DefaultConfig() Config {
	return Config{Enabled: true, TemperatureFreshSeconds: 900, TeslaEnabled: true, TeslaFreshSeconds: 900, TeslaEntities: map[string]string{
		"connected": "binary_sensor.nabijeci_kabel", "soc": "sensor.uroven_baterie", "charging": "sensor.nabijeni",
		"current": "sensor.proud_nabijecky", "power": "sensor.vykon_nabijecky", "current_limit": "number.nabijeci_proud",
	}, Entities: map[string]string{
		"upper": "sensor.tepla_voda", "lower": "sensor.tuv_1", "pump": "switch.kicony_kc868_a16_y04",
		"stage_0": "switch.tuv_0kw", "stage_1": "switch.tuv_1kw", "stage_2": "switch.tuv_2kw", "stage_3": "switch.tuv_3kw",
	}}
}

func Load(path string) (Config, error) {
	c := DefaultConfig()
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, errors.New("nelze číst nastavení HA")
	}
	if json.Unmarshal(b, &c) != nil {
		return c, errors.New("neplatný formát nastavení HA")
	}
	return c, c.Validate()
}

var entityPattern = regexp.MustCompile(`^(sensor|switch|binary_sensor|number)\.[a-z0-9_]+$`)

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.TemperatureFreshSeconds < 30 || c.TemperatureFreshSeconds > 86400 {
		return errors.New("neplatné stáří teplot HA")
	}
	seen := map[string]bool{}
	for key, expected := range DefaultConfig().Entities {
		entity := c.Entities[key]
		if !entityPattern.MatchString(entity) || strings.SplitN(entity, ".", 2)[0] != strings.SplitN(expected, ".", 2)[0] || seen[entity] {
			return errors.New("neplatné nebo duplicitní entity TUV")
		}
		seen[entity] = true
	}
	if len(c.Entities) != 7 {
		return errors.New("neznámá entita TUV")
	}
	if c.TeslaEnabled {
		if c.TeslaFreshSeconds < 30 || c.TeslaFreshSeconds > 86400 {
			return errors.New("neplatné stáří dat Tesly")
		}
		if len(c.TeslaEntities) != 6 {
			return errors.New("neznámá entita Tesly")
		}
		for key, expected := range DefaultConfig().TeslaEntities {
			entity := c.TeslaEntities[key]
			if entity == "" {
				continue
			} // Optional measurements can remain unconfigured.
			if !entityPattern.MatchString(entity) || strings.SplitN(entity, ".", 2)[0] != strings.SplitN(expected, ".", 2)[0] || seen[entity] {
				return errors.New("neplatné nebo duplicitní entity Tesly")
			}
			seen[entity] = true
		}
	}
	return nil
}

type State struct {
	EntityID   string `json:"entity_id"`
	State      string `json:"state"`
	Attributes struct {
		Unit string `json:"unit_of_measurement"`
	} `json:"attributes"`
	LastUpdated  time.Time `json:"last_updated"`
	LastReported time.Time `json:"last_reported"`
}

type Reading struct {
	Value    *float64   `json:"value"`
	Text     *string    `json:"text,omitempty"`
	Unit     string     `json:"unit"`
	Quality  string     `json:"quality"`
	SourceAt *time.Time `json:"source_at"`
}

type Snapshot struct {
	Enabled                 bool               `json:"enabled"`
	Connected               bool               `json:"connected"`
	Status                  string             `json:"status"`
	ReceivedAt              *time.Time         `json:"received_at"`
	TemperatureFreshSeconds int                `json:"temperature_fresh_seconds"`
	Readings                map[string]Reading `json:"readings"`
	NominalPower            Reading            `json:"nominal_power"`
}

type Reader struct {
	mu         sync.Mutex
	config     Config
	token      string
	baseURL    string
	client     *http.Client
	states     map[string]State
	receivedAt time.Time
	status     string
	connected  bool
}

func New(c Config, token string) *Reader {
	status := "connecting"
	if !c.Enabled {
		status = "disabled"
	} else if token == "" {
		status = "no_token"
	}
	return &Reader{config: c, token: token, baseURL: "http://supervisor/core/api/states", status: status,
		client: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}, states: map[string]State{}}
}

func (r *Reader) poll(ctx context.Context) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL, nil)
	if err != nil {
		r.fail("error")
		return
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	resp, err := r.client.Do(req)
	if err != nil {
		r.fail("offline")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		status := "error"
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			status = "unauthorized"
		}
		r.fail(status)
		return
	}
	// Bound the response, discard unselected entities, and never expose raw HA data.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024+1))
	var states []State
	if err != nil || len(body) > 8*1024*1024 || json.Unmarshal(body, &states) != nil || states == nil {
		r.fail("error")
		return
	}
	selected := map[string]State{}
	for _, s := range states {
		for key, entity := range r.config.Entities {
			if s.EntityID == entity {
				selected[key] = s
			}
		}
		if r.config.TeslaEnabled {
			for key, entity := range r.config.TeslaEntities {
				if entity != "" && s.EntityID == entity {
					selected["tesla_"+key] = s
				}
			}
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.states = selected
	r.receivedAt = time.Now()
	r.connected = true
	r.status = "listening"
}

func (r *Reader) fail(status string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = status
	r.connected = false
	r.states = map[string]State{}
	r.receivedAt = time.Time{}
}

func (r *Reader) Start(ctx context.Context) {
	if !r.config.Enabled || r.token == "" {
		return
	}
	go func() {
		r.poll(ctx)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.poll(ctx)
			}
		}
	}()
}

func (r *Reader) Snapshot(now time.Time) Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := Snapshot{Enabled: r.config.Enabled, Connected: r.connected, Status: r.status, TemperatureFreshSeconds: r.config.TemperatureFreshSeconds, Readings: map[string]Reading{}, NominalPower: Reading{Unit: "W", Quality: "missing"}}
	if !r.receivedAt.IsZero() {
		at := r.receivedAt
		out.ReceivedAt = &at
	}
	transportFresh := r.connected && !r.receivedAt.IsZero() && !r.receivedAt.After(now) && now.Sub(r.receivedAt) <= 20*time.Second
	if r.connected && !transportFresh {
		out.Connected = false
		out.Status = "stale"
	}
	for key := range r.config.Entities {
		reading := Reading{Quality: "missing"}
		if key == "upper" || key == "lower" {
			reading.Unit = "°C"
		}
		s, exists := r.states[key]
		if exists {
			at := s.LastReported
			if at.IsZero() {
				at = s.LastUpdated
			}
			if !at.IsZero() {
				reading.SourceAt = &at
			}
			reading.Quality = "invalid"
			if key == "upper" || key == "lower" {
				v, err := strconv.ParseFloat(s.State, 64)
				if err == nil && !math.IsNaN(v) && !math.IsInf(v, 0) && v >= -20 && v <= 150 && s.Attributes.Unit == "°C" {
					reading.Value = &v
					reading.Quality = "valid"
					if at.IsZero() || at.After(now) || now.Sub(at) > time.Duration(r.config.TemperatureFreshSeconds)*time.Second {
						reading.Quality = "stale"
					}
				}
			} else if s.State == "on" || s.State == "off" {
				v := 0.0
				if s.State == "on" {
					v = 1
				}
				reading.Value = &v
				reading.Quality = "valid"
			}
		}
		if !transportFresh && exists {
			reading.Quality = "offline"
		}
		if reading.Quality != "valid" {
			reading.Value = nil
		}
		out.Readings[key] = reading
	}
	count, stage, valid := 0, 0, true
	for i, key := range []string{"stage_0", "stage_1", "stage_2", "stage_3"} {
		s := out.Readings[key]
		if s.Quality != "valid" {
			valid = false
		} else if *s.Value == 1 {
			count++
			stage = i
		}
	}
	if !transportFresh {
		out.NominalPower.Quality = "offline"
	} else if !valid {
		out.NominalPower.Quality = "invalid"
	} else if count != 1 {
		out.NominalPower.Quality = "conflict"
	} else {
		v := float64(stage * 1000)
		out.NominalPower.Value = &v
		out.NominalPower.Quality = "valid"
	}
	return out
}
