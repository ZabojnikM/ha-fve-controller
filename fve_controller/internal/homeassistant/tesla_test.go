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

func teslaFixture(now time.Time) *Reader {
	r := fixture(now, 2)
	for key, entity := range r.config.TeslaEntities {
		parts := strings.SplitN(entity, ".", 2)
		r.config.TeslaEntities[key] = parts[0] + ".test_" + key
		s := State{EntityID: r.config.TeslaEntities[key], LastReported: now}
		switch key {
		case "connected":
			s.State = "on"
		case "charging":
			s.State = "Charging"
		case "soc":
			s.State = "63"
			s.Attributes.Unit = "%"
		case "power":
			s.State = "1.8"
			s.Attributes.Unit = "kW"
		case "current":
			s.State = "8"
			s.Attributes.Unit = "A"
		case "current_limit":
			s.State = "10"
			s.Attributes.Unit = "A"
		}
		r.states["tesla_"+key] = s
	}
	return r
}

func TestTeslaUnitsAndActualVsSetCurrent(t *testing.T) {
	now := time.Now()
	out := teslaFixture(now).TeslaSnapshot(now)
	if *out.Readings["power"].Value != 1800 || out.Readings["power"].Unit != "W" || *out.Readings["current"].Value != 8 || *out.Readings["current_limit"].Value != 10 || *out.Readings["charging"].Text != "charging" {
		t.Fatal("normalization or current distinction failed", out)
	}
	for _, tc := range []struct{ key, state, unit, quality string }{
		{"power", "0", "W", "valid"}, {"current", "0", "A", "valid"}, {"soc", "0", "%", "valid"}, {"soc", "101", "%", "invalid"}, {"current", "-1", "A", "invalid"}, {"power", "2", "VA", "invalid"}, {"power", "NaN", "W", "invalid"}, {"connected", "unknown", "", "invalid"}, {"connected", "off", "", "valid"}, {"charging", "unavailable", "", "invalid"}, {"charging", "NoPower", "", "valid"}, {"charging", "no_power", "", "valid"}, {"charging", "secret-raw-state", "", "invalid"},
	} {
		r := teslaFixture(now)
		s := r.states["tesla_"+tc.key]
		s.State = tc.state
		s.Attributes.Unit = tc.unit
		r.states["tesla_"+tc.key] = s
		reading := r.TeslaSnapshot(now).Readings[tc.key]
		if reading.Quality != tc.quality || (tc.quality != "valid" && (reading.Value != nil || reading.Text != nil)) {
			t.Fatalf("%+v: %+v", tc, reading)
		}
	}
}

func TestTeslaSourceAgeOutageAndOptionalMeasurements(t *testing.T) {
	now := time.Now()
	for _, age := range []time.Duration{48 * time.Hour, -time.Hour, 0} {
		r := teslaFixture(now)
		for key, s := range r.states {
			if strings.HasPrefix(key, "tesla_") {
				s.LastReported = now.Add(-age)
				if age == 0 {
					s.LastReported = time.Time{}
				}
				r.states[key] = s
			}
		}
		for _, reading := range r.TeslaSnapshot(now).Readings {
			if reading.Quality != "valid" || (reading.Value == nil && reading.Text == nil) {
				t.Fatal("available Tesla readings must ignore timestamps", reading)
			}
		}
	}
	r := teslaFixture(now)
	r.config.TeslaEntities["power"] = ""
	delete(r.states, "tesla_current")
	out := r.TeslaSnapshot(now)
	if out.Readings["power"].Quality != "not_configured" || out.Readings["current"].Quality != "missing" || out.Readings["soc"].Quality != "valid" {
		t.Fatal("partial data must remain independent")
	}
	out = r.TeslaSnapshot(now.Add(21 * time.Second))
	if !out.Connected || out.Readings["soc"].Value == nil || out.Readings["charging"].Text == nil {
		t.Fatal("receipt age alone must not hide available state")
	}
	r.fail("offline")
	if r.TeslaSnapshot(now).Readings["connected"].Value != nil {
		t.Fatal("outage must clear cable")
	}
	r.config.TeslaEnabled = false
	if r.TeslaSnapshot(now).Enabled || r.TeslaSnapshot(now).Status != "disabled" {
		t.Fatal("Tesla disabled")
	}
}

func TestSharedPollingDoesNotExposeVehicleData(t *testing.T) {
	now := time.Now()
	r := teslaFixture(now)
	states := []State{}
	for _, s := range r.states {
		states = append(states, s)
	}
	states = append(states, State{EntityID: "device_tracker.private_location", State: "private-location"})
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests++
		if req.Method != "POST" || req.URL.Path != "/template" {
			t.Error("unexpected control or wake request")
		}
		var payload struct {
			Template  string              `json:"template"`
			Variables map[string][]string `json:"variables"`
		}
		if json.NewDecoder(req.Body).Decode(&payload) != nil || payload.Template != stateTemplate || req.Header.Get("Content-Type") != "application/json" {
			t.Error("expected fixed read-only template")
		}
		entities := payload.Variables["entity_ids"]
		if len(entities) != 14 {
			t.Error("request must select only TUV and Tesla entities")
		}
		for _, entity := range entities {
			if entity == "device_tracker.private_location" || entity == "" {
				t.Error("request includes unselected entity")
			}
		}
		json.NewEncoder(w).Encode(states)
	}))
	defer server.Close()
	r.baseURL = server.URL + "/template"
	r.poll(context.Background())
	if requests != 1 || len(r.states) != 14 || r.Snapshot(time.Now()).NominalPower.Quality != "valid" || r.TeslaSnapshot(time.Now()).Readings["soc"].Quality != "valid" {
		t.Fatal("TUV and Tesla must share one polling cycle")
	}
	b, _ := json.Marshal(r.TeslaSnapshot(time.Now()))
	for _, private := range []string{"test-secret", "private-location", "private_location", "test_connected"} {
		if strings.Contains(string(b), private) {
			t.Fatal("private data leaked")
		}
	}
	r.fail("offline")
	r.poll(context.Background())
	if r.TeslaSnapshot(time.Now()).Readings["soc"].Quality != "valid" {
		t.Fatal("fresh recovery failed")
	}
}

func TestUnchangedCurrentUsesLiveReportTime(t *testing.T) {
	now := time.Now().UTC()
	r := teslaFixture(now)
	s := r.states["tesla_current"]
	s.State = "16"
	s.LastUpdated = now.Add(-40 * time.Minute)
	s.LastReported = now.Add(-5 * time.Second)
	code := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(code)
		json.NewEncoder(w).Encode([]State{s})
	}))
	defer server.Close()
	r.baseURL = server.URL + "/template"
	r.poll(context.Background())
	reading := r.TeslaSnapshot(time.Now()).Readings["current"]
	if reading.Quality != "valid" || reading.Value == nil || *reading.Value != 16 || !reading.SourceAt.Equal(s.LastReported) {
		t.Fatal("unchanged current with fresh report must stay valid", reading)
	}
	// Available values remain valid even when the source report time is old.
	s.LastReported = now.Add(-20 * time.Minute)
	r.poll(context.Background())
	if reading := r.TeslaSnapshot(time.Now()).Readings["current"]; reading.Quality != "valid" || reading.Value == nil || *reading.Value != 16 {
		t.Fatal("old report must remain usable when HA provides the value", reading)
	}
	// A rejected template endpoint clears the previous snapshot; no /states fallback.
	code = http.StatusForbidden
	r.poll(context.Background())
	if r.status != "unauthorized" || len(r.states) != 0 {
		t.Fatal("template access denial must clear state")
	}
}

func TestExistingOptionsUpgradeAndTeslaConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.json")
	if err := os.WriteFile(path, []byte(`{"mqtt_enabled":true,"ha_enabled":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil || !c.TeslaEnabled || len(c.TeslaEntities) != 6 || c.Validate() != nil {
		t.Fatal("old options must gain defaults", err)
	}
	c.TeslaEntities["power"] = ""
	if c.Validate() != nil {
		t.Fatal("optional entity")
	}
	c.TeslaEntities["current"] = "sensor.test/../services"
	if c.Validate() == nil {
		t.Fatal("unsafe entity")
	}
}
