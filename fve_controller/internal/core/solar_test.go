package core

import (
	"math"
	"testing"
	"time"
)

func TestSolarSum(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name    string
		value   float64
		valid   bool
		age     time.Duration
		unit    string
		quality string
		want    float64
	}{
		{"production", 600, true, 0, "W", "valid", 4600},
		{"zero string", 0, true, 0, "W", "valid", 4000},
		{"invalid", 600, false, 0, "W", "invalid", 0},
		{"stale", 600, true, time.Minute, "W", "stale", 0},
		{"future", 600, true, -time.Minute, "W", "invalid", 0},
		{"negative", -1, true, 0, "W", "invalid", 0},
		{"nan", math.NaN(), true, 0, "W", "invalid", 0},
		{"wrong unit", 600, true, 0, "kW", "invalid", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inputs := [3]Sample{{Value: 2600, Unit: "W", At: now, Valid: true}, {Value: 1400, Unit: "W", At: now, Valid: true}, {Value: tc.value, Unit: tc.unit, At: now.Add(-tc.age), Valid: tc.valid}}
			got := SolarSum(now, inputs)
			if got.Quality(now) != tc.quality || got.Value != tc.want {
				t.Fatalf("unexpected total: %+v quality=%s", got, got.Quality(now))
			}
		})
	}
	zero := Sample{Unit: "W", At: now, Valid: true}
	if got := SolarSum(now, [3]Sample{zero, zero, zero}); got.Quality(now) != "valid" || got.Value != 0 {
		t.Fatalf("night must be valid zero: %+v", got)
	}
	if got := SolarSum(now, [3]Sample{zero, zero, {}}); got.Quality(now) != "invalid" {
		t.Fatal("missing input must invalidate total")
	}
}
