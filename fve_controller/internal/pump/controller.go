// Package pump owns only the TUV mixing pump. Heating and other outputs remain
// outside its scope. Runtime state is rebuilt from fresh HA readings on restart.
package pump

import (
	"context"
	"encoding/json"
	"errors"
	"fve-controller/internal/homeassistant"
	"math"
	"os"
	"sync"
	"time"
	_ "time/tzdata"
)

type Config struct {
	Enabled      bool `json:"pump_control_enabled"`
	FreshSeconds int  `json:"pump_temperature_fresh_seconds"`
}

func Load(path string) (Config, error) {
	c := Config{FreshSeconds: 120}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return c, nil
	}
	c.Enabled = true
	if err != nil || json.Unmarshal(b, &c) != nil {
		return c, errors.New("Nelze číst nastavení čerpadla")
	}
	if c.FreshSeconds < 30 || c.FreshSeconds > 900 {
		return c, errors.New("Neplatné stáří teplot pro čerpadlo")
	}
	return c, nil
}

type Sender interface {
	Set(context.Context, bool) error
}
type Snapshot struct {
	Ready          bool       `json:"ready"`
	Mode           string     `json:"mode"`
	ManualRequest  *bool      `json:"manual_request"`
	Enabled        bool       `json:"enabled"`
	Owner          string     `json:"owner"`
	Status         string     `json:"status"`
	Reason         string     `json:"reason"`
	Desired        *bool      `json:"desired"`
	Sent           *bool      `json:"sent"`
	Confirmed      *bool      `json:"confirmed"`
	SentAt         *time.Time `json:"sent_at"`
	Error          string     `json:"error,omitempty"`
	Process        bool       `json:"process_request"`
	Service        bool       `json:"service_request"`
	ServiceUntil   *time.Time `json:"service_until"`
	LastServiceDay string     `json:"last_service_day"`
	FreshSeconds   int        `json:"temperature_fresh_seconds"`
}

type Controller struct {
	mu              sync.Mutex
	config          Config
	sender          Sender
	persistDay      func(string) error
	location        *time.Location
	view            Snapshot
	process         bool
	initialized     bool
	serviceDay      string
	service         bool
	serviceUntil    time.Time
	serviceDeadline time.Time
	pending         bool
	sent            bool
	sentAt          time.Time
	retryAt         time.Time
	lastCycle       time.Time
	hasAttempt      bool
	lastAttempt     bool
	serviceRetryAt  time.Time
	serviceError    string
	mode            string
	manualRequest   bool
	healthy         bool
}

func New(c Config, sender Sender, lastDay string, persistDay func(string) error) *Controller {
	loc, _ := time.LoadLocation("Europe/Prague") // Embedded tzdata also works in Alpine.
	owner, status, reason := "external", "not_owned", "Řízení čerpadla není předané doplňku"
	if c.Enabled {
		owner, status, reason = "addon", "waiting", "Čeká na živý stav čerpadla a teploty"
	}
	return &Controller{config: c, sender: sender, persistDay: persistDay, location: loc, serviceDay: lastDay, mode: "auto",
		view: Snapshot{Mode: "auto", Enabled: c.Enabled, Owner: owner, Status: status, Reason: reason, LastServiceDay: lastDay, FreshSeconds: c.FreshSeconds}}
}

func boolValue(v bool) *bool           { return &v }
func timeValue(v time.Time) *time.Time { return &v }

func temperature(r homeassistant.Reading, now time.Time, ttl int) (float64, bool) {
	if r.Quality != "valid" || r.Value == nil || r.Unit != "°C" || r.SourceAt == nil || r.SourceAt.After(now) || now.Sub(*r.SourceAt) > time.Duration(ttl)*time.Second {
		return 0, false
	}
	v := *r.Value
	return v, !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 100
}

func actual(s homeassistant.Snapshot, now time.Time) *bool {
	r := s.Readings["pump"]
	if !s.Enabled || !s.Connected || s.ReceivedAt == nil || s.ReceivedAt.After(now) || now.Sub(*s.ReceivedAt) > 20*time.Second || r.Quality != "valid" || r.Value == nil || (*r.Value != 0 && *r.Value != 1) {
		return nil
	}
	return boolValue(*r.Value == 1)
}

// Cycle is called serially by Run; only Snapshot is read concurrently by HTTP.
// Manual requests are accepted only through the central TUV coordinator.
func (c *Controller) Cycle(ctx context.Context, now time.Time, s homeassistant.Snapshot) {
	if ctx.Err() != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.view.Confirmed = actual(s, now)
	if !c.config.Enabled {
		return
	}
	c.view.Error = c.serviceError
	if !c.lastCycle.IsZero() && now.Before(c.lastCycle) {
		c.process, c.service = false, false
		c.initialized = false
		c.pending = false
		c.retryAt = time.Time{}
	}
	c.lastCycle = now
	upper, upperOK := temperature(s.Readings["upper"], now, c.config.FreshSeconds)
	lower, lowerOK := temperature(s.Readings["lower"], now, c.config.FreshSeconds)
	reset := s.Readings["sensor_reset"]
	resetOK := reset.Quality == "valid" && reset.Value != nil && *reset.Value == 0
	healthy := c.view.Confirmed != nil && upperOK && lowerOK && resetOK
	c.healthy = healthy
	c.view.Ready = healthy
	if !healthy {
		c.manualRequest = false
		c.process, c.service = false, false
		c.serviceUntil = time.Time{}
		c.view.Reason = "Chybí platná čerstvá teplota nebo stav čerpadla/napájení čidel"
		if reset.Quality == "valid" && reset.Value != nil && *reset.Value == 1 {
			c.view.Reason = "Obnova čidel · požadováno vypnutí čerpadla"
		}
	} else if c.mode == "manual" {
		c.process, c.service = false, false
		c.serviceUntil = time.Time{}
		c.view.Reason = "Ruční ovládání · automatické promíchávání i protočení jsou vypnuté"
	} else {
		if !c.initialized {
			// Fresh feedback seeds the hysteresis, never persisted commands.
			c.process = *c.view.Confirmed
			c.initialized = true
		}
		if upper < 57 || lower >= 57 {
			c.process = false
		} else if upper > 58 && lower < 57 {
			c.process = true
		}
		local := now.In(c.location)
		day := local.Format("2006-01-02")
		// Do not replay missed services after a restart or late recovery.
		if local.Hour() == 18 && local.Minute() == 30 && c.serviceDay != day && (c.serviceDay == "" || day > c.serviceDay) && !now.Before(c.serviceRetryAt) {
			if c.persistDay == nil || c.persistDay(day) != nil {
				c.serviceError = "Servisní protočení nelze zaznamenat"
				c.view.Error = c.serviceError
				c.serviceRetryAt = now.Add(10 * time.Second)
			} else {
				c.serviceError, c.view.Error = "", ""
				c.serviceDay = day
				c.service = true
				c.serviceUntil = time.Time{}
				c.serviceDeadline = time.Date(local.Year(), local.Month(), local.Day(), 18, 31, 0, 0, c.location)
			}
		}
		if c.service && c.serviceUntil.IsZero() {
			if *c.view.Confirmed {
				c.serviceUntil = now.Add(30 * time.Second)
			} else if !now.Before(c.serviceDeadline) {
				c.service = false
			}
		}
		if c.service && !c.serviceUntil.IsZero() && !now.Before(c.serviceUntil) {
			c.service = false
		}
		c.view.Reason = "Teploty nevyžadují promíchávání"
		if c.process {
			c.view.Reason = "Promíchávání · horní vrstva teplá, spodní chladnější"
		}
		if c.service {
			c.view.Reason = "Denní servisní protočení"
			if c.process {
				c.view.Reason = "Promíchávání a servisní protočení"
			}
		}
	}
	c.view.Process, c.view.Service, c.view.LastServiceDay = c.process, c.service, c.serviceDay
	c.view.ServiceUntil = nil
	if c.service && !c.serviceUntil.IsZero() {
		c.view.ServiceUntil = timeValue(c.serviceUntil)
	}
	desired := healthy && (c.process || c.service)
	if c.mode == "manual" {
		desired = healthy && c.manualRequest
		c.view.ManualRequest = boolValue(c.manualRequest)
	} else {
		c.view.ManualRequest = nil
	}
	c.view.Desired = boolValue(desired)
	if c.view.Confirmed == nil {
		c.initialized = false
		c.view.Status = "unavailable"
		// An OFF request is useful even when the selected entity cannot be read;
		// startup with no HA snapshot must not issue any commands.
		if !s.Enabled || !s.Connected || s.ReceivedAt == nil {
			return
		}
	}
	if c.pending && c.sent == desired {
		if c.view.Confirmed != nil && *c.view.Confirmed == desired && s.ReceivedAt.After(c.sentAt) {
			c.pending = false
		} else if now.Sub(c.sentAt) < 15*time.Second {
			if c.view.Confirmed != nil {
				c.view.Status = "waiting_confirmation"
			}
			return
		} else {
			c.pending = false
			c.retryAt = now.Add(10 * time.Second)
			c.view.Status, c.view.Error = "confirmation_timeout", "HA nepotvrdil změnu čerpadla"
			return
		}
	}
	if c.view.Confirmed != nil && *c.view.Confirmed == desired && (!c.pending || c.sent == desired) {
		c.pending = false
		c.view.Status = "idle"
		if !healthy {
			c.view.Status = "input_invalid"
		} else if c.serviceError != "" {
			c.view.Status = "service_error"
		}
		return
	}
	// A change of target (particularly ON -> OFF) supersedes pending commands.
	changed := c.hasAttempt && c.lastAttempt != desired
	if !changed && now.Before(c.retryAt) {
		c.view.Status, c.view.Error = "retry_wait", "Čeká na další pokus po chybě nebo nepotvrzeném povelu"
		return
	}
	c.view.Status = "sending"
	c.hasAttempt, c.lastAttempt = true, desired
	if c.sender == nil || c.sender.Set(ctx, desired) != nil {
		c.pending = false
		c.retryAt = now.Add(10 * time.Second)
		c.view.Status, c.view.Error = "command_error", "Povel čerpadla se nepodařilo odeslat"
		return
	}
	c.sent, c.sentAt, c.pending = desired, now, true
	c.view.Sent, c.view.SentAt = boolValue(desired), timeValue(now)
	c.retryAt = now.Add(10 * time.Second)
	c.view.Status = "waiting_confirmation"
	if c.view.Confirmed == nil {
		c.view.Status = "unavailable"
	}
}

func (c *Controller) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.view
}

// Stop makes a best-effort OFF request during an orderly shutdown. It does not
// claim physical confirmation, and cannot cover power loss or a killed process.
func (c *Controller) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.config.Enabled || (!c.initialized && !c.hasAttempt) || c.sender == nil {
		return nil
	}
	return c.sender.Set(ctx, false)
}

func (c *Controller) Run(ctx context.Context, read func(time.Time) homeassistant.Snapshot) {
	if !c.config.Enabled {
		return
	}
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				c.Cycle(ctx, now, read(now))
			}
		}
	}()
}

// SetMode is called only by the central TUV coordinator after persistence.
func (c *Controller) SetMode(mode string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.mode == mode {
		return
	}
	c.mode, c.view.Mode = mode, mode
	c.manualRequest, c.process, c.service, c.initialized = false, false, false, false
	c.view.Desired = nil
	c.serviceUntil = time.Time{}
	c.view.Process, c.view.Service, c.view.ServiceUntil = false, false, nil
	c.view.ManualRequest = nil
	if mode == "manual" {
		c.view.ManualRequest = boolValue(false)
	}
	if c.config.Enabled {
		c.view.Status = "waiting"
		c.view.Reason = "Přepíná režim TUV podle živých stavů"
	}
}
func (c *Controller) Manual(on bool, now time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.config.Enabled || c.mode != "manual" {
		return errors.New("Čerpadlo není předané ručnímu ovládání doplňku")
	}
	if on && (!c.healthy || now.Before(c.lastCycle) || now.Sub(c.lastCycle) > 3*time.Second) {
		return errors.New("Zapnutí čerpadla vyžaduje čerstvé platné vstupy")
	}
	c.manualRequest = on
	c.view.ManualRequest = boolValue(on)
	return nil
}
