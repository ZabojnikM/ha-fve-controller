package homeassistant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixture(now time.Time, stage int) *Reader {
	c := DefaultConfig()
	// Anonymous IDs in test fixtures, unrelated to the installation.
	for key := range c.Entities {
		c.Entities[key] = strings.SplitN(c.Entities[key], ".", 2)[0] + ".test_" + key
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
		} else if key == "power" {
			s.State = []string{"Vypnuto", "1 kW", "2 kW", "3 kW"}[stage]
		} else if key == "system" {
			s.State = "Aktivní"
		} else if key == "uptime" {
			s.State = "3600"
			s.Attributes.Unit = "s"
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
	for _, state := range []string{"unknown", "unavailable", "", "0", "2", "2kW", "4 kW", "on"} {
		r := fixture(now, 2)
		s := r.states["power"]
		s.State = state
		r.states["power"] = s
		out := r.Snapshot(now)
		if out.NominalPower.Value != nil || out.NominalPower.Quality != "invalid" {
			t.Fatalf("%s: %+v", state, out.NominalPower)
		}
	}
	r := fixture(now, 2)
	delete(r.states, "power")
	if r.Snapshot(now).NominalPower.Value != nil || r.Snapshot(now).NominalPower.Quality != "missing" {
		t.Fatal("missing select must not confirm power")
	}
}

func TestTemperatureQualityAndTransport(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		state, unit string
		age         time.Duration
		quality     string
	}{
		{"0", "°C", 0, "valid"}, {"48", "°C", time.Hour, "valid"}, {"48", "°C", -time.Hour, "valid"}, {"unavailable", "°C", 0, "invalid"}, {"unknown", "°C", 0, "invalid"}, {"NaN", "°C", 0, "invalid"}, {"48", "°F", 0, "invalid"}, {"999", "°C", 0, "invalid"},
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
	s = r.states["power"]
	s.LastReported = now.Add(-24 * time.Hour)
	r.states["power"] = s
	if r.Snapshot(now).NominalPower.Quality != "valid" {
		t.Fatal("stable switch state should remain readable")
	}
	out := r.Snapshot(now.Add(21 * time.Second))
	if !out.Connected || out.NominalPower.Value == nil || out.Readings["upper"].Value == nil {
		t.Fatal("receipt age must not hide available values")
	}
	r.fail("offline")
	if r.Snapshot(now).NominalPower.Value != nil {
		t.Fatal("outage must clear state")
	}
	if New(DefaultConfig(), "test-secret").Snapshot(now).NominalPower.Value != nil {
		t.Fatal("restart must wait for new snapshot")
	}
}

func TestTemperatureWithoutReportTimeAndLegacyOptions(t *testing.T) {
	now := time.Now()
	r := fixture(now, 0)
	for _, key := range []string{"upper", "lower"} {
		s := r.states[key]
		s.LastReported, s.LastUpdated = time.Time{}, time.Time{}
		r.states[key] = s
		if value := r.Snapshot(now).Readings[key]; value.Quality != "valid" || value.Value == nil {
			t.Fatal("available numeric temperature needs no report timestamp", key, value)
		}
	}
	path := filepath.Join(t.TempDir(), "options.json")
	if err := os.WriteFile(path, []byte(`{"ha_temperature_fresh_seconds":1,"pump_temperature_fresh_seconds":1,"tesla_fresh_seconds":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err != nil {
		t.Fatal("obsolete temperature age options must not prevent startup", err)
	}
	r.receivedAt = now.Add(-21 * time.Second)
	if value := r.Snapshot(now).Readings["upper"]; value.Quality != "valid" || value.Value == nil {
		t.Fatal("receipt age must not invalidate available temperature", value)
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
	if len(r.states) != 8 || r.Snapshot(time.Now()).NominalPower.Quality != "valid" {
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
	c.Entities["sensor_reset"] = c.Entities["pump"]
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
	if reading := r.Snapshot(time.Now()).Readings["lower"]; reading.Quality != "valid" || reading.Value == nil || *reading.Value != 35.3 {
		t.Fatal("unchanged numeric temperature must stay valid regardless of report age", reading)
	}
}

func TestKiconyBlocksFreshnessAndIndependentInputs(t *testing.T) {
	now := time.Now()
	for _, state := range []string{"Obnova čidel TUV", "Zablokováno (Teplota)", "Zablokováno (Porucha čidel po 3 resetech)", "Zablokováno (Porucha čidla)", "Zablokováno (Watchdog)", "Zablokováno (Přetížení)", "Aktivní", "unknown", "unavailable", "private-state"} {
		r := fixture(now, 0)
		s := r.states["system"]
		s.State = state
		r.states["system"] = s
		out := r.Snapshot(now)
		invalid := state == "unknown" || state == "unavailable" || state == "private-state"
		if (out.Readings["system"].Text == nil) != invalid || out.NominalPower.Quality != "valid" || *out.NominalPower.Value != 0 {
			t.Fatal("block text is independent of a valid zero power", out)
		}
	}
	for _, key := range []string{"system", "uptime"} {
		for _, age := range []time.Duration{4 * time.Minute, -time.Second} {
			r := fixture(now, 2)
			s := r.states[key]
			s.LastReported = now.Add(-age)
			r.states[key] = s
			v := r.Snapshot(now).Readings[key]
			if v.Quality != "valid" || (v.Text == nil && v.Value == nil) {
				t.Fatal("available periodic state must ignore report time", v)
			}
		}
	}
	r := fixture(now, 2)
	for _, key := range []string{"overload", "sensor_reset"} {
		s := r.states[key]
		s.State = "on"
		s.LastReported = now.Add(-time.Hour)
		r.states[key] = s
	}
	out := r.Snapshot(now)
	if *out.Readings["overload"].Value != 1 || *out.Readings["sensor_reset"].Value != 1 || *out.Readings["system"].Text != "Aktivní" {
		t.Fatal("inputs must remain independent of textual report", out)
	}
	r.fail("offline")
	out = r.Snapshot(now.Add(21 * time.Second))
	for _, v := range out.Readings {
		if v.Value != nil || v.Text != nil {
			t.Fatal("transport error must clear all visible values", v)
		}
	}
}

func TestLegacyTuvOptionsMigrateWithoutChangingExternalTemperature(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.json")
	legacy := `{"ha_enabled":true,"ha_entities":{"upper":"sensor.test_external_water","lower":"sensor.test_lower","pump":"switch.test_pump","stage_0":"switch.test_old_0","stage_1":"switch.test_old_1","stage_2":"switch.test_old_2","stage_3":"switch.test_old_3"}}`
	if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil || len(c.Entities) != 8 || c.Entities["upper"] != "sensor.test_external_water" || c.Entities["pump"] != "switch.test_pump" || c.Entities["power"] != DefaultConfig().Entities["power"] {
		t.Fatal("migration must retain custom mappings and add select", c.Entities, err)
	}
	for key, entity := range c.Entities {
		if strings.HasPrefix(key, "stage_") || entity == "sensor.tuv_2" {
			t.Fatal("old switches and TUV2 must not be read")
		}
	}
	// An explicitly supplied invalid select must not be silently replaced.
	if err := os.WriteFile(path, []byte(`{"ha_entities":{"power":""}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("explicit empty mapping must fail validation")
	}
}

func TestAllHAReadingsIgnoreTimestampsButKeepAvailability(t *testing.T) {
	now := time.Now()
	for _, stamp := range []time.Time{time.Time{}, now.Add(-48 * time.Hour), now.Add(time.Hour)} {
		r := teslaFixture(now)
		r.receivedAt = stamp
		for key, state := range r.states {
			state.LastReported, state.LastUpdated = stamp, stamp
			r.states[key] = state
		}
		for _, out := range []map[string]Reading{r.Snapshot(now).Readings, r.TeslaSnapshot(now).Readings} {
			for key, reading := range out {
				if reading.Quality != "valid" || (reading.Value == nil && reading.Text == nil) {
					t.Fatal("HA timestamps must not expire available data", key, reading)
				}
			}
		}
		for key, s := range r.states {
			s.State = "unavailable"
			r.states[key] = s
		}
		for _, out := range []map[string]Reading{r.Snapshot(now).Readings, r.TeslaSnapshot(now).Readings} {
			for _, reading := range out {
				if reading.Quality != "invalid" || reading.Value != nil || reading.Text != nil {
					t.Fatal("unavailable values must remain hidden", reading)
				}
			}
		}
		r.fail("offline")
		if r.Snapshot(now).Connected || r.TeslaSnapshot(now).Connected || len(r.states) != 0 {
			t.Fatal("communication error must clear both snapshots")
		}
	}
}
