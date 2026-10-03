// Package tuv coordinates the central mode for the mixing pump only.
// Heating remains exclusively with Node-RED; no heating sender exists here.
package tuv

import (
	"errors"
	"fve-controller/internal/pump"
	"sync"
	"time"
)

type Snapshot struct {
	Mode             string `json:"mode"`
	PumpOwned        bool   `json:"pump_owned"`
	AutomaticHeating bool   `json:"automatic_heating"`
	HeatingOwner     string `json:"heating_owner"`
}
type Controller struct {
	mu      sync.Mutex
	pump    *pump.Controller
	persist func(string) error
	view    Snapshot
}

func New(p *pump.Controller, mode string, save func(string) error) *Controller {
	if mode != "manual" {
		mode = "auto"
	}
	p.SetMode(mode) // Manual requests always start OFF, never restored from storage.
	return &Controller{pump: p, persist: save, view: Snapshot{Mode: mode, PumpOwned: p.Snapshot().Enabled, HeatingOwner: "node_red"}}
}
func (c *Controller) Snapshot() Snapshot { c.mu.Lock(); defer c.mu.Unlock(); return c.view }
func (c *Controller) SetMode(mode string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if mode != "auto" && mode != "manual" {
		return errors.New("Neplatný režim TUV")
	}
	if !c.view.PumpOwned {
		return errors.New("Čerpadlo není předané doplňku")
	}
	if c.view.Mode == mode {
		return nil
	}
	if c.persist == nil || c.persist(mode) != nil {
		return errors.New("Režim TUV se nepodařilo uložit")
	}
	c.pump.SetMode(mode)
	c.view.Mode = mode
	return nil
}
func (c *Controller) ManualPump(on bool, now time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.view.Mode != "manual" {
		return errors.New("Nejdřív zvolte Ruční ovládání")
	}
	return c.pump.Manual(on, now)
}
