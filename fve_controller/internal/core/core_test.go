package core

import (
	"math"
	"testing"
	"time"
)

func fixture(n time.Time) Input {
	s := func(v float64) Sample { return Sample{Value: v, At: n, Valid: true, Source: "fixture"} }
	return Input{SOC: s(80), Battery: s(2500), Temperature: s(48), MinCell: s(3.44), MaxCell: s(3.48), TUV: s(1000), Car: s(0), Connected: true, Tariff: "VT"}
}
func engine(n time.Time) Engine { return Engine{LastBalance: n} }
func TestQuality(t *testing.T) {
	n := time.Now()
	for _, tc := range []struct {
		name string
		s    Sample
		want string
	}{{"zero", Sample{Valid: true, At: n}, "valid"}, {"missing", Sample{}, "invalid"}, {"old", Sample{Valid: true, At: n.Add(-16 * time.Second)}, "stale"}, {"nan", Sample{Valid: true, At: n, Value: math.NaN()}, "invalid"}, {"future", Sample{Valid: true, At: n.Add(time.Second)}, "stale"}} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.Quality(n); got != tc.want {
				t.Fatalf("%s != %s", got, tc.want)
			}
		})
	}
}
func TestBatterySignAndPriority(t *testing.T) {
	n := time.Now()
	in := fixture(n)
	e := engine(n)
	d := e.Step(n, in)
	if d.TUV != 2000 || d.Car != 0 {
		t.Fatalf("gradual TUV priority: %+v", d)
	}
	e = engine(n)
	in.Battery.Value = -1500
	d = e.Step(n, in)
	if d.TUV+d.Car != 0 {
		t.Fatal("discharge must reduce loads")
	}
	e = engine(n)
	in.Battery.Value = 2500
	in.Temperature.Value = 67
	d = e.Step(n, in)
	if d.TUV != 0 || d.Car != 1150 {
		t.Fatalf("hot TUV releases allocation: %+v", d)
	}
}
func TestHighHysteresisAndProbe(t *testing.T) {
	n := time.Now()
	e := engine(n)
	for _, tc := range []struct{ soc, target float64 }{{95, -300}, {94, -300}, {93, 0}, {94, 0}} {
		in := fixture(n)
		in.SOC.Value = tc.soc
		d := e.Step(n, in)
		if d.Target != tc.target {
			t.Fatalf("SOC %v: %v", tc.soc, d.Target)
		}
	}
	e = engine(n)
	in := fixture(n)
	in.SOC.Value = 95
	in.Battery.Value = 0
	in.TUV.Value = 0
	d := e.Step(n, in)
	if d.TUV != 1000 {
		t.Fatal("zero flow needs probe")
	}
	d = e.Step(n.Add(time.Second), in)
	if d.TUV != 0 {
		t.Fatal("must wait for response")
	}
}
func TestProtectionModes(t *testing.T) {
	n := time.Now()
	e := engine(n)
	for _, tc := range []struct {
		soc          float64
		tariff, mode string
	}{{9, "VT", "SÍŤ + NABÍJENÍ"}, {12, "VT", "SÍŤ + NABÍJENÍ"}, {13, "VT", "SÍŤ"}, {19, "VT", "SÍŤ"}, {20, "VT", "FVE"}, {15, "NT", "SÍŤ + NABÍJENÍ"}, {16, "NT", "SÍŤ + NABÍJENÍ"}, {17, "NT", "FVE"}} {
		in := fixture(n)
		in.SOC.Value = tc.soc
		in.Tariff = tc.tariff
		if d := e.Step(n, in); d.Mode != tc.mode {
			t.Fatalf("%+v: %+v", tc, d)
		}
	}
}
func TestInvalidOverloadManual(t *testing.T) {
	n := time.Now()
	for _, kind := range []string{"invalid", "stale", "overload", "manual-hot"} {
		e := engine(n)
		in := fixture(n)
		in.Manual = true
		in.ManualWatts = 3000
		switch kind {
		case "invalid":
			in.SOC.Valid = false
		case "stale":
			in.Battery.At = n.Add(-time.Minute)
		case "overload":
			in.Overload = true
		case "manual-hot":
			in.Temperature.Value = 66
		}
		d := e.Step(n, in)
		if d.TUV != 0 {
			t.Fatalf("%s: %+v", kind, d)
		}
	}
}
func TestBalanceContinuousAndRestart(t *testing.T) {
	n := time.Now()
	e := Engine{}
	for i := 0; i <= 300; i += 5 {
		now := n.Add(time.Duration(i) * time.Second)
		in := fixture(now)
		in.SOC.Value = 95
		d := e.Step(now, in)
		if d.Target != 300 {
			t.Fatal("balance overrides negative target")
		}
		if i < 300 && !e.LastBalance.IsZero() {
			t.Fatal("premature completion")
		}
	}
	if e.LastBalance.IsZero() {
		t.Fatal("must complete")
	}
	e = Engine{}
	for i := 0; i <= 300; i += 5 {
		now := n.Add(time.Duration(i) * time.Second)
		in := fixture(now)
		in.SOC.Value = 95
		if i == 150 {
			in.MinCell.Valid = false
		}
		e.Step(now, in)
	}
	if !e.LastBalance.IsZero() {
		t.Fatal("invalid sample must reset hold")
	}
	e = Engine{}
	in := fixture(n)
	in.SOC.Value = 95
	e.Step(n, in)
	later := n.Add(6 * time.Minute)
	in = fixture(later)
	in.SOC.Value = 95
	e.Step(later, in)
	if !e.LastBalance.IsZero() {
		t.Fatal("gap must reset hold")
	}
	restarted := Engine{LastBalance: n}
	if !restarted.heldSince.IsZero() || restarted.High {
		t.Fatal("restart must not restore live state")
	}
}
func TestCommandLifecycle(t *testing.T) {
	n := time.Now()
	var tr Tracker
	if tr.Confirmed != nil {
		t.Fatal("startup unknown")
	}
	tr.SendSimulation(1000, n)
	old := tr.ID
	tr.Check(n.Add(11 * time.Second))
	if tr.Status != "timeout" {
		t.Fatal("timeout")
	}
	if tr.Ack(old, 1000, n.Add(12*time.Second)) {
		t.Fatal("late ack")
	}
	tr.SendSimulation(2000, n.Add(20*time.Second))
	if tr.Ack(old, 1000, n.Add(21*time.Second)) {
		t.Fatal("old ack")
	}
	if !tr.Ack(tr.ID, 2000, n.Add(22*time.Second)) {
		t.Fatal("valid ack")
	}
}

func TestSharedLimitAndBalanceWaiting(t *testing.T) {
	n := time.Now()
	e := engine(n)
	in := fixture(n)
	in.TUV.Value = 3000
	in.Car.Value = 1150
	in.Battery.Value = 3000
	d := e.Step(n, in)
	if d.TUV != 3000 || d.Car == 0 || d.TUV+d.Car > 4600 {
		t.Fatalf("shared allocation: %+v", d)
	}
	e = Engine{High: true}
	in.SOC.Value = 94
	d = e.Step(n, in)
	if d.Target < 0 {
		t.Fatal("balance waiting must suppress discharge target")
	}
	tr := Tracker{}
	tr.SendSimulation(1000, n)
	tr.Fail(tr.ID)
	if tr.Status != "error" || tr.Ack(tr.ID, 1000, n) {
		t.Fatal("command failure is not waiting")
	}
}

func TestBalanceRequestEdgeAndInterval(t *testing.T) {
	n := time.Now()
	e := engine(n)
	for i := 0; i <= 305; i += 5 {
		now := n.Add(time.Duration(i) * time.Second)
		in := fixture(now)
		in.SOC.Value = 95
		in.BalanceRequest = true
		e.Step(now, in)
	}
	if e.Balancing {
		t.Fatal("held request must not restart completed balance")
	}
	now := e.LastBalance.Add(14 * 24 * time.Hour)
	in := fixture(now)
	in.SOC.Value = 95
	e.Step(now, in)
	if !e.Balancing {
		t.Fatal("14 day interval must trigger")
	}
}
