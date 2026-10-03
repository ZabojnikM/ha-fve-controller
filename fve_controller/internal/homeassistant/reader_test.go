package homeassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func fixture(now time.Time, stage int) *Reader {
	c := DefaultConfig()
	// Anonymous IDs in test fixtures, unrelated to the installation.
	for key := range c.Entities {
		if key == "upper" || key == "lower" {
			c.Entities[key] = "sensor.test_" + key
		} else {
			c.Entities[key] = "switch.test_" + key
		}
	}
	r := New(c, "test-secret")
	r.connected = true
	r.status = "listening"
	r.receivedAt = now
	for key, entity := range c.Entities {
		s := State{EntityID: entity, State: "off", LastReported: now}
		if key == "upper" || key == "lower" {
			s.State = "48.5"
			s.Attributes.Unit = "°C"
		} else if key == fmt.Sprintf("stage_%d", stage) {
			s.State = "on"
		}
		r.states[key] = s
	}
	return r
}

func TestStagesAndInvalidFeedback(t *testing.T) {
	now := time.Now()
	for stage := 0; stage <= 3; stage++ {
		s := fixture(now, stage).Snapshot(now)
		if s.NominalPower.Quality != "valid" || *s.NominalPower.Value != float64(stage*1000) {
			t.Fatalf("stage %d: %+v", stage, s)
		}
	}
	for _, change := range []struct {
		key, state string
		quality    string
	}{
		{"stage_1", "on", "conflict"}, {"stage_2", "off", "conflict"}, {"stage_0", "unavailable", "invalid"}, {"stage_0", "unknown", "invalid"}, {"stage_0", "", "invalid"},
	} {
		r := fixture(now, 2)
		s := r.states[change.key]
		s.State = change.state
		r.states[change.key] = s
		out := r.Snapshot(now)
		if out.NominalPower.Value != nil || out.NominalPower.Quality != change.quality {
			t.Fatalf("%+v: %+v", change, out.NominalPower)
		}
	}
	r := fixture(now, 2)
	delete(r.states, "stage_0")
	if r.Snapshot(now).NominalPower.Value != nil {
		t.Fatal("missing switch must not confirm power")
	}
}

func TestTemperatureQualityAndTransport(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		state, unit string
		age         time.Duration
		quality     string
	}{
		{"0", "°C", 0, "valid"}, {"48", "°C", time.Hour, "stale"}, {"48", "°C", -time.Hour, "stale"}, {"unknown", "°C", 0, "invalid"}, {"NaN", "°C", 0, "invalid"}, {"48", "°F", 0, "invalid"}, {"999", "°C", 0, "invalid"},
	} {
		r := fixture(now, 2)
		s := r.states["upper"]
		s.State = tc.state
		s.Attributes.Unit = tc.unit
		s.LastReported = now.Add(-tc.age)
		r.states["upper"] = s
		out := r.Snapshot(now).Readings["upper"]
		if out.Quality != tc.quality || (tc.quality != "valid" && out.Value != nil) {
			t.Fatalf("%+v: %+v", tc, out)
		}
	}
	r := fixture(now, 2)
	s := r.states["upper"]
	s.LastReported = time.Time{}
	s.LastUpdated = now
	r.states["upper"] = s
	if r.Snapshot(now).Readings["upper"].Quality != "valid" {
		t.Fatal("last_updated fallback")
	}
	// Unchanged switch timestamps do not imply a dead connection.
	s = r.states["stage_2"]
	s.LastReported = now.Add(-24 * time.Hour)
	r.states["stage_2"] = s
	if r.Snapshot(now).NominalPower.Quality != "valid" {
		t.Fatal("stable switch state should remain readable")
	}
	out := r.Snapshot(now.Add(21 * time.Second))
	if out.Connected || out.NominalPower.Value != nil || out.Readings["upper"].Value != nil {
		t.Fatal("transport silence must hide all values")
	}
	r.fail("offline")
	if r.Snapshot(now).NominalPower.Value != nil {
		t.Fatal("outage must clear state")
	}
	if New(DefaultConfig(), "test-secret").Snapshot(now).NominalPower.Value != nil {
		t.Fatal("restart must wait for new snapshot")
	}
}

func TestReadOnlyPollingAndPrivacy(t *testing.T) {
	now := time.Now()
	r := fixture(now, 2)
	states := []State{}
	for _, s := range r.states {
		states = append(states, s)
	}
	states = append(states, State{EntityID: "sensor.private_unselected", State: "private-state"})
	code := 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" || req.URL.Path != "/template" || req.Header.Get("Authorization") != "Bearer test-secret" {
			t.Error("unexpected request")
		}
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(states)
	}))
	defer server.Close()
	r.baseURL = server.URL + "/template"
	r.poll(context.Background())
	if len(r.states) != 7 || r.Snapshot(time.Now()).NominalPower.Quality != "valid" {
		t.Fatal("poll did not select expected states")
	}
	b, _ := json.Marshal(r.Snapshot(time.Now()))
	for _, secret := range []string{"test-secret", "private-state", "private_unselected", "test_stage"} {
		if strings.Contains(string(b), secret) {
			t.Fatal("API leaks private state or configuration")
		}
	}
	code = 401
	r.poll(context.Background())
	if r.status != "unauthorized" || len(r.states) != 0 || r.Snapshot(time.Now()).NominalPower.Value != nil {
		t.Fatal("auth failure must clear state")
	}
	code = 200
	r.poll(context.Background())
	if r.Snapshot(time.Now()).NominalPower.Quality != "valid" {
		t.Fatal("recovery needs fresh data")
	}
}

func TestConfigAndNoToken(t *testing.T) {
	c := DefaultConfig()
	if c.Validate() != nil {
		t.Fatal("default config invalid")
	}
	c.Entities["pump"] = "switch.test/../services"
	if c.Validate() == nil {
		t.Fatal("invalid entity accepted")
	}
	c = DefaultConfig()
	c.Entities["stage_1"] = c.Entities["stage_0"]
	if c.Validate() == nil {
		t.Fatal("duplicate feedback accepted")
	}
	r := New(DefaultConfig(), "")
	r.Start(context.Background())
	if r.Snapshot(time.Now()).Status != "no_token" {
		t.Fatal("missing token must be explicit")
	}
}

func TestUnchangedTemperatureUsesLiveReportTime(t *testing.T) {
	now := time.Now().UTC()
	r := fixture(now, 0)
	s := r.states["lower"]
	s.State = "35.3"
	s.LastUpdated = now.Add(-30 * time.Minute)
	s.LastReported = now.Add(-5 * time.Second)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode([]State{s})
	}))
	defer server.Close()
	r.baseURL = server.URL + "/template"
	r.poll(context.Background())
	reading := r.Snapshot(time.Now()).Readings["lower"]
	if reading.Quality != "valid" || reading.Value == nil || *reading.Value != 35.3 || !reading.SourceAt.Equal(s.LastReported) {
		t.Fatal("unchanged temperature with a fresh report must stay valid", reading)
	}
	s.LastReported = now.Add(-20 * time.Minute)
	r.poll(context.Background())
	if reading := r.Snapshot(time.Now()).Readings["lower"]; reading.Quality != "stale" || reading.Value != nil {
		t.Fatal("fresh HTTP receipt must not validate an old temperature report", reading)
	}
}
