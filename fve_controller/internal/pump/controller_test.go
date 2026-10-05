package pump

import (
	"context"
	"errors"
	"fve-controller/internal/homeassistant"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type sender struct {
	calls []bool
	fail  bool
}

func (s *sender) Set(_ context.Context, on bool) error {
	s.calls = append(s.calls, on)
	if s.fail {
		return errors.New("test failure")
	}
	return nil
}
func fixture(now time.Time, upper, lower float64, on bool) homeassistant.Snapshot {
	p := 0.0
	if on {
		p = 1
	}
	r := func(v float64, unit string) homeassistant.Reading {
		return homeassistant.Reading{Value: &v, Unit: unit, Quality: "valid", SourceAt: &now}
	}
	return homeassistant.Snapshot{Enabled: true, Connected: true, ReceivedAt: &now, Readings: map[string]homeassistant.Reading{
		"upper": r(upper, "°C"), "lower": r(lower, "°C"), "pump": r(p, ""), "sensor_reset": r(0, ""),
	}}
}
func controller(s *sender) *Controller {
	return New(Config{Enabled: true}, s, "", func(string) error { return nil })
}
func tick(c *Controller, now time.Time, upper, lower float64, on bool) {
	c.Cycle(context.Background(), now, fixture(now, upper, lower, on))
}
func noon() time.Time { t, _ := time.Parse(time.RFC3339, "2026-10-04T12:00:00+02:00"); return t }

func TestProcessHysteresisAndExactBoundaries(t *testing.T) {
	for _, tc := range []struct {
		upper, lower  float64
		current, want bool
	}{
		{58, 56, false, false}, {58.01, 56, false, true}, {57, 56, true, true}, {57.9, 56, false, false},
		{56.99, 56, true, false}, {59, 57, true, false}, {59, 57, false, false}, {59, 56.99, false, true},
		{0, 0, true, false}, {0, 0, false, false}, {65, 40, false, true},
	} {
		s := &sender{}
		c := controller(s)
		tick(c, noon(), tc.upper, tc.lower, tc.current)
		out := c.Snapshot()
		if out.Desired == nil || *out.Desired != tc.want || out.Process != tc.want {
			t.Fatalf("%+v: %+v", tc, out)
		}
		if tc.want == tc.current && len(s.calls) != 0 {
			t.Fatal("unchanged state must not switch relay")
		}
	}
}

func TestDisabledStartupInvalidInputsAndOutage(t *testing.T) {
	s := &sender{}
	c := New(Config{}, s, "", nil)
	tick(c, noon(), 59, 40, false)
	if len(s.calls) != 0 || c.Snapshot().Owner != "external" {
		t.Fatal("unowned output must never receive commands")
	}
	c = controller(s)
	c.Cycle(context.Background(), noon(), homeassistant.Snapshot{})
	if len(s.calls) != 0 || c.Snapshot().Confirmed != nil {
		t.Fatal("startup must wait for live state")
	}
	for _, change := range []string{"missing", "invalid", "unavailable", "reset"} {
		s = &sender{}
		c = controller(s)
		now := noon()
		in := fixture(now, 59, 40, true)
		switch change {
		case "missing":
			delete(in.Readings, "upper")
		case "invalid":
			r := in.Readings["upper"]
			r.Value = nil
			in.Readings["upper"] = r
		case "unavailable":
			r := in.Readings["upper"]
			r.Value = nil
			r.Quality = "invalid"
			in.Readings["upper"] = r
		case "reset":
			r := in.Readings["sensor_reset"]
			v := 1.0
			r.Value = &v
			in.Readings["sensor_reset"] = r
		}
		c.Cycle(context.Background(), now, in)
		if *c.Snapshot().Desired || c.Snapshot().Process {
			t.Fatal("bad input must not enable pump", change)
		}
		if len(s.calls) != 1 || s.calls[0] {
			t.Fatal("bad input should request OFF only", change, s.calls)
		}
	}
	s = &sender{}
	c = controller(s)
	tick(c, noon(), 59, 40, false)
	in := fixture(noon().Add(time.Second), 59, 40, true)
	in.Connected = false
	c.Cycle(context.Background(), noon().Add(time.Second), in)
	if c.Snapshot().Confirmed != nil || c.Snapshot().Status != "unavailable" {
		t.Fatal("outage must not claim stopped pump")
	}
}

func TestAcknowledgmentTimeoutRetriesAndLateOldConfirmation(t *testing.T) {
	s := &sender{}
	c := controller(s)
	now := noon()
	tick(c, now, 59, 40, false)
	if c.Snapshot().Status != "waiting_confirmation" || *c.Snapshot().Confirmed {
		t.Fatal("HTTP acceptance is not confirmation")
	}
	// Same receipt cannot acknowledge even an expected output value.
	in := fixture(now, 59, 40, true)
	c.Cycle(context.Background(), now.Add(time.Second), in)
	if c.Snapshot().Status != "waiting_confirmation" {
		t.Fatal("pre-command read must not acknowledge command")
	}
	tick(c, now.Add(5*time.Second), 59, 40, true)
	if c.Snapshot().Status != "idle" || len(s.calls) != 1 {
		t.Fatal("fresh confirmation should settle without reissuing")
	}
	// Cooling overrides the old ON even if its feedback is delayed.
	tick(c, now.Add(6*time.Second), 56, 40, true)
	if len(s.calls) != 2 || s.calls[1] {
		t.Fatal("cooling must send OFF")
	}
	tick(c, now.Add(7*time.Second), 59, 40, false) // New thermal ON supersedes OFF.
	if len(s.calls) != 3 || !s.calls[2] {
		t.Fatal("new target must supersede pending old OFF")
	}
	tick(c, now.Add(8*time.Second), 59, 40, false) // Delayed old OFF cannot confirm new ON.
	if c.Snapshot().Status != "waiting_confirmation" || len(s.calls) != 3 {
		t.Fatal("late old confirmation must not settle new target")
	}
	tick(c, now.Add(22*time.Second), 59, 40, false)
	if c.Snapshot().Status != "confirmation_timeout" {
		t.Fatal("missing feedback should become timeout")
	}
	tick(c, now.Add(23*time.Second), 59, 40, false)
	if len(s.calls) != 3 {
		t.Fatal("timeout must back off")
	}
	tick(c, now.Add(32*time.Second), 59, 40, false)
	if len(s.calls) != 4 {
		t.Fatal("unconfirmed target must be retried")
	}
}

func TestCommandErrorsBackOffButSafetyOffPreemptsOn(t *testing.T) {
	s := &sender{fail: true}
	c := controller(s)
	now := noon()
	tick(c, now, 59, 40, false)
	if c.Snapshot().Status != "command_error" || c.Snapshot().Sent != nil {
		t.Fatal("failed send must remain unaccepted")
	}
	tick(c, now.Add(time.Second), 59, 40, false)
	if len(s.calls) != 1 {
		t.Fatal("error must not cause per-cycle retries")
	}
	tick(c, now.Add(10*time.Second), 59, 40, false)
	if len(s.calls) != 2 {
		t.Fatal("failed command must retry after backoff")
	}
	s.fail = false
	tick(c, now.Add(20*time.Second), 59, 40, false)
	s.fail = true
	tick(c, now.Add(21*time.Second), 56, 40, true)
	if len(s.calls) != 4 || s.calls[3] {
		t.Fatal("safety OFF must preempt pending ON")
	}
	tick(c, now.Add(22*time.Second), 56, 40, true)
	if len(s.calls) != 4 {
		t.Fatal("failed OFF must also back off")
	}
}

func TestDailyServiceThirtyConfirmedSecondsAndOR(t *testing.T) {
	start, _ := time.Parse(time.RFC3339, "2026-10-04T18:30:00+02:00")
	for _, process := range []bool{false, true} {
		s := &sender{}
		saved := []string{}
		c := New(Config{Enabled: true}, s, "", func(day string) error { saved = append(saved, day); return nil })
		upper := 50.0
		if process {
			upper = 59
		}
		tick(c, start, upper, 40, false)
		if !c.Snapshot().Service || len(saved) != 1 || len(s.calls) != 1 || !s.calls[0] {
			t.Fatal("daily service must reserve day then request ON")
		}
		ack := start.Add(5 * time.Second)
		tick(c, ack, upper, 40, true)
		if c.Snapshot().ServiceUntil == nil || !c.Snapshot().ServiceUntil.Equal(ack.Add(30*time.Second)) {
			t.Fatal("count service from read confirmation")
		}
		tick(c, ack.Add(29*time.Second), upper, 40, true)
		if !c.Snapshot().Service || len(s.calls) != 1 {
			t.Fatal("service must not end early")
		}
		tick(c, ack.Add(30*time.Second), upper, 40, true)
		if c.Snapshot().Service || *c.Snapshot().Desired != process {
			t.Fatal("service expiration must preserve process request")
		}
		if !process && (len(s.calls) != 2 || s.calls[1]) {
			t.Fatal("service-only request must end with OFF")
		}
		if process && len(s.calls) != 1 {
			t.Fatal("service expiration must not interrupt thermal mixing")
		}
		// Restart after reservation never restarts or restores a service timer.
		restarted := New(Config{Enabled: true}, s, saved[0], func(string) error { return nil })
		tick(restarted, start.Add(40*time.Second), 50, 40, true)
		if restarted.Snapshot().Service || *restarted.Snapshot().Desired {
			t.Fatal("restart must cancel service and use live temperatures")
		}
	}
}

func TestServiceClockTimezoneSkippedDaysAndStorageFailure(t *testing.T) {
	for _, stamp := range []string{"2026-10-04T16:30:00Z", "2026-10-26T17:30:00Z"} { // Prague before/after DST.
		now, _ := time.Parse(time.RFC3339, stamp)
		s := &sender{}
		c := controller(s)
		tick(c, now, 50, 40, false)
		if !c.Snapshot().Service {
			t.Fatal("service must use Prague timezone", stamp)
		}
	}
	s := &sender{}
	c := controller(s)
	now, _ := time.Parse(time.RFC3339, "2026-10-04T18:32:00+02:00")
	tick(c, now, 50, 40, false)
	if len(s.calls) != 0 || c.Snapshot().Service {
		t.Fatal("missed schedule must not be replayed")
	}
	c = New(Config{Enabled: true}, s, "", func(string) error { return errors.New("storage failed") })
	now = now.Add(-2 * time.Minute)
	tick(c, now, 50, 40, false)
	if len(s.calls) != 0 || c.Snapshot().Service || c.Snapshot().Error == "" {
		t.Fatal("failed durable reservation must not start service")
	}
	// Clock going backwards cannot preserve an old service lease.
	c = controller(s)
	tick(c, now, 50, 40, false)
	tick(c, now.Add(-time.Minute), 50, 40, true)
	if c.Snapshot().Service || *c.Snapshot().Desired {
		t.Fatal("clock rewind must clear service")
	}
}

func TestConfigAndOrderlyStop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.json")
	cfg, err := Load(path)
	if err != nil || cfg.Enabled {
		t.Fatal("default must not claim ownership")
	}
	if err := os.WriteFile(path, []byte(`{"pump_control_enabled":true,"pump_temperature_fresh_seconds":5}`), 0600); err != nil {
		t.Fatal(err)
	}
	if cfg, err := Load(path); err != nil || !cfg.Enabled {
		t.Fatal("legacy temperature age option must be ignored")
	}
	s := &sender{}
	c := controller(s)
	if c.Stop(context.Background()) != nil || len(s.calls) != 0 {
		t.Fatal("uninitialized stop must not control pump")
	}
	tick(c, noon(), 59, 40, true)
	if c.Stop(context.Background()) != nil || len(s.calls) != 1 || s.calls[0] {
		t.Fatal("orderly stop should request OFF")
	}
}

func TestManualSuppressesAutomationAndRestartClearsRequest(t *testing.T) {
	s := &sender{}
	c := controller(s)
	now := noon()
	c.SetMode("manual")
	tick(c, now, 59, 40, false)
	if len(s.calls) != 0 || c.Snapshot().Process {
		t.Fatal("manual must ignore thermal ON")
	}
	if err := c.Manual(true, now); err != nil {
		t.Fatal(err)
	}
	tick(c, now.Add(time.Second), 59, 40, false)
	if len(s.calls) != 1 || !s.calls[0] || *c.Snapshot().Confirmed {
		t.Fatal("manual request is not confirmation")
	}
	tick(c, now.Add(2*time.Second), 59, 40, true)
	if c.Snapshot().Status != "idle" {
		t.Fatal("fresh feedback must confirm")
	}
	if err := c.Manual(false, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	tick(c, now.Add(3*time.Second), 59, 40, true)
	if len(s.calls) != 2 || s.calls[1] {
		t.Fatal("manual OFF must preempt hot water")
	}
	restart := controller(s)
	restart.SetMode("manual")
	if restart.Snapshot().Sent != nil || restart.Snapshot().Confirmed != nil {
		t.Fatal("restart restores no commands or feedback")
	}
	tick(restart, now.Add(4*time.Second), 59, 40, true)
	if *restart.Snapshot().Desired || *restart.Snapshot().ManualRequest {
		t.Fatal("restart manual request must be OFF")
	}
	start, _ := time.Parse(time.RFC3339, "2026-10-04T18:30:00+02:00")
	tick(restart, start, 59, 40, false)
	if restart.Snapshot().Service || restart.Snapshot().Process {
		t.Fatal("manual must suppress service and process")
	}
	restart.SetMode("auto")
	tick(restart, start.Add(time.Second), 59, 40, false)
	if !*restart.Snapshot().Desired {
		t.Fatal("return to auto evaluates current inputs")
	}
}

func TestManualRejectsUnsafeOnAndAllowsOff(t *testing.T) {
	s := &sender{}
	c := controller(s)
	now := noon()
	c.SetMode("manual")
	if c.Manual(true, now) == nil {
		t.Fatal("startup ON must wait for live data")
	}
	tick(c, now, 59, 40, false)
	if c.Manual(true, now.Add(4*time.Second)) != nil {
		t.Fatal("valid available inputs do not expire by age")
	}
	in := fixture(now.Add(time.Second), 59, 40, true)
	delete(in.Readings, "upper")
	c.Cycle(context.Background(), now.Add(time.Second), in)
	if c.Manual(true, now.Add(time.Second)) == nil {
		t.Fatal("invalid temperature cannot authorize ON")
	}
	if c.Manual(false, now.Add(time.Second)) != nil {
		t.Fatal("OFF must remain available")
	}
}

func TestModeChangeCancelsServiceWithoutRestoringIt(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2026-10-04T18:30:00+02:00")
	s := &sender{}
	c := controller(s)
	tick(c, now, 50, 40, false)
	c.SetMode("manual")
	tick(c, now.Add(time.Second), 50, 40, true)
	if c.Snapshot().Service || *c.Snapshot().Desired || len(s.calls) != 2 || s.calls[1] {
		t.Fatal("manual cancels service with OFF")
	}
	c.SetMode("auto")
	tick(c, now.Add(2*time.Second), 50, 40, false)
	if c.Snapshot().Service {
		t.Fatal("same service day cannot replay after mode change")
	}
}

func TestOptionsReadyDefaultPreservesExplicitDisable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.json")
	for _, tc := range []struct {
		body    string
		enabled bool
	}{
		{`{}`, true}, {`{"pump_control_enabled":false}`, false}, {`{"pump_control_enabled":true}`, true},
	} {
		if err := os.WriteFile(path, []byte(tc.body), 0600); err != nil {
			t.Fatal(err)
		}
		c, err := Load(path)
		if err != nil || c.Enabled != tc.enabled {
			t.Fatal(tc, err, c)
		}
	}
}

func TestTemperatureReportTimeDoesNotBlockControl(t *testing.T) {
	now := noon()
	for _, report := range []*time.Time{nil, timeValue(now.Add(-48 * time.Hour)), timeValue(now.Add(time.Hour))} {
		for _, mode := range []string{"auto", "manual"} {
			s := &sender{}
			c := controller(s)
			c.SetMode(mode)
			in := fixture(now, 59, 40, false)
			in.ReceivedAt = report
			for _, key := range []string{"upper", "lower"} {
				r := in.Readings[key]
				r.SourceAt = report
				in.Readings[key] = r
			}
			c.Cycle(context.Background(), now, in)
			if !c.Snapshot().Ready {
				t.Fatal("numeric available temperatures must not require report timestamps", mode, report)
			}
			if mode == "manual" {
				if err := c.Manual(true, now); err != nil {
					t.Fatal(err)
				}
				c.Cycle(context.Background(), now.Add(time.Second), in)
			}
			if len(s.calls) != 1 || !s.calls[0] {
				t.Fatal("valid unchanged temperatures must allow ON", mode, s.calls)
			}
			// Availability remains required even when temperature age is ignored.
			r := in.Readings["lower"]
			r.Value, r.Quality = nil, "invalid"
			in.Readings["lower"] = r
			c.Cycle(context.Background(), now.Add(2*time.Second), in)
			if c.Snapshot().Ready || *c.Snapshot().Desired || len(s.calls) != 2 || s.calls[1] {
				t.Fatal("unavailable temperature must cancel ON", mode, s.calls)
			}
		}
	}
}
