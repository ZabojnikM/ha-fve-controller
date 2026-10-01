package core

import (
	"math"
	"time"
)

type Sample struct {
	Value  float64   `json:"value"`
	Unit   string    `json:"unit"`
	At     time.Time `json:"at"`
	Valid  bool      `json:"valid"`
	Source string    `json:"source"`
}

func (s Sample) Quality(now time.Time) string {
	if !s.Valid || math.IsNaN(s.Value) || math.IsInf(s.Value, 0) {
		return "invalid"
	}
	if s.At.IsZero() || now.Sub(s.At) > 15*time.Second || s.At.After(now) {
		return "stale"
	}
	return "valid"
}

type Input struct {
	SOC            Sample  `json:"soc"`
	Battery        Sample  `json:"battery"`
	Temperature    Sample  `json:"temperature"`
	MinCell        Sample  `json:"min_cell"`
	MaxCell        Sample  `json:"max_cell"`
	TUV            Sample  `json:"tuv"`
	Car            Sample  `json:"car"`
	Overload       bool    `json:"overload"`
	Connected      bool    `json:"connected"`
	Tariff         string  `json:"tariff"`
	Manual         bool    `json:"manual"`
	ManualWatts    float64 `json:"manual_watts"`
	BalanceRequest bool    `json:"balance_request"`
}
type Decision struct {
	Mode        string    `json:"mode"`
	Reason      string    `json:"reason"`
	Target      float64   `json:"target"`
	TUV         float64   `json:"tuv"`
	Car         float64   `json:"car"`
	Balance     string    `json:"balance"`
	LastBalance time.Time `json:"last_balance"`
	NextBalance time.Time `json:"next_balance"`
}
type Engine struct {
	High, Critical, Protection, Balancing          bool
	previousBalanceRequest                         bool
	LastBalance, heldSince, lastTick, lastIncrease time.Time
}

func (e *Engine) Step(now time.Time, in Input) Decision {
	d := Decision{Mode: "FVE", Reason: "Společný rozpočet • přednost TUV", Balance: "Čeká na termín", LastBalance: e.LastBalance, NextBalance: e.LastBalance.Add(14 * 24 * time.Hour)}
	if e.LastBalance.IsZero() || now.Sub(e.LastBalance) >= 14*24*time.Hour || (in.BalanceRequest && !e.previousBalanceRequest) {
		e.Balancing = true
	}
	e.previousBalanceRequest = in.BalanceRequest
	if now.Sub(e.lastTick) > 15*time.Second {
		e.heldSince = time.Time{}
	}
	e.lastTick = now
	for _, s := range []Sample{in.SOC, in.Battery, in.Temperature, in.TUV, in.Car} {
		if s.Quality(now) != "valid" {
			e.heldSince = time.Time{}
			d.Mode = "BLOKACE"
			d.Reason = "Neplatný nebo zastaralý vstup — čeká na čerstvý stav"
			d.Balance = "Potvrzování přerušeno"
			return d
		}
	}
	if in.SOC.Value < 0 || in.SOC.Value > 100 || in.TUV.Value < 0 || in.Car.Value < 0 || in.Temperature.Value < 0 || in.Temperature.Value > 110 {
		e.heldSince = time.Time{}
		d.Mode = "BLOKACE"
		d.Reason = "Vstup mimo fyzikální rozsah"
		return d
	}
	if in.Overload {
		e.heldSince = time.Time{}
		d.Mode = "BLOKACE"
		d.Reason = "Přetížení — odlehčit spotřebiče"
		return d
	}
	if in.SOC.Value >= 95 {
		e.High = true
	}
	if in.SOC.Value <= 93 {
		e.High = false
	}
	if in.SOC.Value < 10 {
		e.Critical = true
	}
	if e.Critical {
		if in.SOC.Value < 13 {
			d.Mode = "SÍŤ + NABÍJENÍ"
			d.Reason = "Kritická ochrana • dobití na 13 %"
			e.heldSince = time.Time{}
			return d
		}
		e.Critical = false
		e.Protection = true
	}
	if in.SOC.Value < 16 {
		e.Protection = true
	}
	if e.Protection {
		exit := 20.0
		if in.Tariff == "NT" {
			exit = 17
		}
		if in.SOC.Value < exit {
			d.Mode = "SÍŤ"
			if in.Tariff == "NT" {
				d.Mode = "SÍŤ + NABÍJENÍ"
			}
			d.Reason = "Ochranná hystereze baterie"
			e.heldSince = time.Time{}
			return d
		}
		e.Protection = false
	}
	if e.High {
		d.Target = -300
	}
	if e.Balancing {
		d.Target = 0
		d.Balance = "Čeká na SOC ≥ 95 %"
		if in.SOC.Value >= 95 {
			d.Target = 300
			d.Balance = "Rezerva +300 W • čeká na články"
			if in.MinCell.Quality(now) == "valid" && in.MinCell.Value >= 3.439 && in.MinCell.Value <= 3.7 {
				if e.heldSince.IsZero() {
					e.heldSince = now
				}
				d.Balance = "Potvrzuje napětí po dobu 5 minut"
				if now.Sub(e.heldSince) >= 5*time.Minute {
					e.LastBalance = now
					e.Balancing = false
					e.heldSince = time.Time{}
					d.Balance = "Dokončeno"
					d.LastBalance = now
					d.NextBalance = now.Add(14 * 24 * time.Hour)
				}
			} else {
				e.heldSince = time.Time{}
				d.Balance = "Čeká na platné napětí ≥ 3,439 V"
			}
		} else {
			e.heldSince = time.Time{}
		}
	}
	budget := math.Max(0, math.Min(4600, in.TUV.Value+in.Car.Value+in.Battery.Value-d.Target))
	tuvMax := 3000.0
	if in.Temperature.Value >= 60 && !in.Manual || in.Temperature.Value > 65 {
		tuvMax = 0
		d.Reason = "Teplotní blokace TUV • rozpočet pro Teslu"
	}
	if in.Manual {
		tuvMax = math.Min(tuvMax, math.Max(0, math.Min(3000, math.Floor(in.ManualWatts/1000)*1000)))
		d.Reason = "Ruční požadavek TUV • ochrany zůstávají aktivní"
	}
	d.TUV = math.Min(tuvMax, math.Floor(budget/1000)*1000)
	// Discrete 1 kW probe at curtailed MPPT; never during balancing.
	if e.High && !e.Balancing && in.TUV.Value+in.Car.Value == 0 && budget > 0 && budget < 1000 && tuvMax >= 1000 {
		d.TUV = 1000
		budget = 1000
		d.Reason = "Průzkumný krok TUV 1 kW • vysoký SOC"
	}
	left := math.Max(0, budget-d.TUV)
	if in.Connected && left >= 5*230 {
		d.Car = math.Min(16, math.Floor(left/230)) * 230
	}
	current := in.TUV.Value + in.Car.Value
	if d.TUV+d.Car > current {
		if now.Sub(e.lastIncrease) < 10*time.Second {
			d.TUV = math.Min(d.TUV, in.TUV.Value)
			d.Car = math.Min(d.Car, in.Car.Value)
			d.Reason = "Čeká na odezvu • prodleva 10 s"
		} else {
			// Only one load increases per tick; each change is one physical step.
			if d.TUV > in.TUV.Value {
				d.TUV = math.Min(d.TUV, in.TUV.Value+1000)
				d.Car = math.Min(d.Car, in.Car.Value)
			} else if d.Car > in.Car.Value {
				step := 230.0
				if in.Car.Value == 0 {
					step = 1150
				}
				d.Car = math.Min(d.Car, in.Car.Value+step)
			}
			e.lastIncrease = now
		}
	}
	return d
}

// Tracker is simulation-only. No physical command transport exists.
type Tracker struct {
	Desired   float64   `json:"desired"`
	Sent      *float64  `json:"sent"`
	Confirmed *float64  `json:"confirmed"`
	Status    string    `json:"status"`
	ID        uint64    `json:"id"`
	SentAt    time.Time `json:"-"`
}

func (t *Tracker) SendSimulation(w float64, now time.Time) {
	t.ID++
	t.Desired = w
	t.Sent = &w
	t.SentAt = now
	t.Status = "waiting"
}
func (t *Tracker) Ack(id uint64, w float64, now time.Time) bool {
	if t.Status != "waiting" || id != t.ID || now.Sub(t.SentAt) > 10*time.Second || t.Sent == nil || w != *t.Sent {
		return false
	}
	t.Confirmed = &w
	t.Status = "confirmed"
	return true
}
func (t *Tracker) Check(now time.Time) {
	if t.Status == "waiting" && now.Sub(t.SentAt) > 10*time.Second {
		t.Status = "timeout"
	}
}

func (t *Tracker) Fail(id uint64) {
	if t.ID == id && t.Status == "waiting" {
		t.Status = "error"
	}
}
