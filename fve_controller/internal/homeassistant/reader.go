// Package homeassistant reads HA state only; it never calls services or writes states.
package homeassistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"os"
	"regexp"
	"sort"
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
		"power": "select.automatizace_kicony_vykon_tuv", "system": "sensor.stav_systemu_tuv",
		"overload": "binary_sensor.sonoff_4ch_button_1", "sensor_reset": "switch.kicony_kc868_a16_y15", "uptime": "sensor.esp_tuv_cas_behu",
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
	// Older options gain the new defaults, retaining custom temperature/pump IDs.
	// Removed optimistic switches must never be used as a power fallback.
	for _, key := range []string{"stage_0", "stage_1", "stage_2", "stage_3"} {
		delete(c.Entities, key)
	}
	return c, c.Validate()
}

var entityPattern = regexp.MustCompile(`^(sensor|switch|binary_sensor|number|select)\.[a-z0-9_]+$`)

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
	if len(c.Entities) != len(DefaultConfig().Entities) {
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

// Read current State properties directly: /states serves cached serialized states
// whose last_reported may remain frozen while unchanged values are reported.
// Entity IDs are variables, never executable template text. No services are called.
const stateTemplate = `{% set ns = namespace(items=[]) %}{% for entity_id in entity_ids %}{% set s = states[entity_id] %}{% if s is not none %}{% set ns.items = ns.items + [{'entity_id': entity_id, 'state': s.state, 'attributes': {'unit_of_measurement': s.attributes.get('unit_of_measurement', '')}, 'last_updated': s.last_updated.isoformat(), 'last_reported': s.last_reported.isoformat() if s.last_reported is defined else s.last_updated.isoformat()}] %}{% endif %}{% endfor %}{{ ns.items | to_json }}`

func (r *Reader) templateRequest() []byte {
	selected := map[string]bool{}
	for _, entity := range r.config.Entities {
		if entity != "" {
			selected[entity] = true
		}
	}
	if r.config.TeslaEnabled {
		for _, entity := range r.config.TeslaEntities {
			if entity != "" {
				selected[entity] = true
			}
		}
	}
	entities := make([]string, 0, len(selected))
	for entity := range selected {
		entities = append(entities, entity)
	}
	sort.Strings(entities)
	// This payload contains only strings and slices, so JSON encoding cannot fail.
	body, _ := json.Marshal(struct {
		Template  string              `json:"template"`
		Variables map[string][]string `json:"variables"`
	}{stateTemplate, map[string][]string{"entity_ids": entities}})
	return body
}

func New(c Config, token string) *Reader {
	status := "connecting"
	if !c.Enabled {
		status = "disabled"
	} else if token == "" {
		status = "no_token"
	}
	return &Reader{config: c, token: token, baseURL: "http://supervisor/core/api/template", status: status,
		client: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}, states: map[string]State{}}
}

func (r *Reader) poll(ctx context.Context) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL, bytes.NewReader(r.templateRequest()))
	if err != nil {
		r.fail("error")
		return
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/json")
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
			} else if key == "power" {
				if v, ok := map[string]float64{"Vypnuto": 0, "1 kW": 1000, "2 kW": 2000, "3 kW": 3000}[s.State]; ok {
					reading.Value = &v
					reading.Unit = "W"
					reading.Quality = "valid"
				}
			} else if key == "system" {
				// Only documented states are exposed, never arbitrary raw HA text.
				switch s.State {
				case "Obnova čidel TUV", "Zablokováno (Teplota)", "Zablokováno (Porucha čidel po 3 resetech)", "Zablokováno (Porucha čidla)", "Zablokováno (Watchdog)", "Zablokováno (Přetížení)", "Aktivní":
					text := s.State
					reading.Text = &text
					reading.Quality = "valid"
				}
				if reading.Quality == "valid" && (at.IsZero() || at.After(now) || now.Sub(at) > 60*time.Second) {
					reading.Quality = "stale"
				}
			} else if key == "uptime" {
				v, err := strconv.ParseFloat(s.State, 64)
				if err == nil && !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && s.Attributes.Unit == "s" {
					reading.Value = &v
					reading.Unit = "s"
					reading.Quality = "valid"
					if at.IsZero() || at.After(now) || now.Sub(at) > 180*time.Second {
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
			reading.Text = nil
		}
		out.Readings[key] = reading
	}
	out.NominalPower = out.Readings["power"]
	out.NominalPower.Unit = "W"
	if !transportFresh {
		out.NominalPower.Quality = "offline"
	}
	return out
}
