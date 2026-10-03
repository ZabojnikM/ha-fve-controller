package homeassistant

import (
	"testing"
	"time"
)

func TestInverterPowerValidityAndFreshness(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		state, unit, quality string
		age                  time.Duration
		value                float64
	}{
		{"0", "W", "valid", 0, 0}, {"3.5", "kW", "valid", 0, 3500}, {"8200", "W", "valid", 0, 8200},
		{"-1", "W", "invalid", 0, 0}, {"NaN", "W", "invalid", 0, 0}, {"unknown", "W", "invalid", 0, 0},
		{"2", "A", "invalid", 0, 0}, {"100", "W", "stale", 61 * time.Second, 0}, {"100", "W", "stale", -time.Second, 0},
	} {
		t.Run(tc.state+tc.unit+tc.age.String(), func(t *testing.T) {
			r := New(DefaultConfig(), "test-secret")
			s := State{State: tc.state, LastUpdated: now.Add(-time.Hour), LastReported: now.Add(-tc.age)}
			s.Attributes.Unit = tc.unit
			r.states["inverter"] = s
			r.connected = true
			r.receivedAt = now
			got := r.PowerSnapshot(now).Readings["inverter"]
			if got.Quality != tc.quality {
				t.Fatalf("quality %s", got.Quality)
			}
			if tc.quality == "valid" {
				if got.Value == nil || *got.Value != tc.value {
					t.Fatal("normalized value")
				}
			} else if got.Value != nil {
				t.Fatal("invalid value exposed")
			}
			if r.PowerSnapshot(now.Add(21 * time.Second)).Readings["inverter"].Value != nil {
				t.Fatal("transport outage must hide value")
			}
			r.fail("unauthorized")
			if r.PowerSnapshot(now).Readings["inverter"].Value != nil {
				t.Fatal("authorization failure must clear state")
			}
		})
	}
	r := New(DefaultConfig(), "")
	if r.PowerSnapshot(now).Readings["inverter"].Quality != "missing" {
		t.Fatal("missing is not zero")
	}
}
