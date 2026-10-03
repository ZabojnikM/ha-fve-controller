package main

import (
	"database/sql"
	"encoding/json"
	"fve-controller/internal/homeassistant"
	"fve-controller/internal/pump"
	"fve-controller/internal/tuv"
	"fve-controller/internal/victron"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTUVAPIIngressAndPrivacy(t *testing.T) {
	a := &App{ha: homeassistant.New(homeassistant.DefaultConfig(), "test-secret")}
	h := a.handler(t.TempDir(), true)
	for _, peer := range []string{"1.2.3.4:1000", "172.30.32.2:1000"} {
		r := httptest.NewRequest("GET", "/api/tuv", nil)
		r.RemoteAddr = peer
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if peer == "1.2.3.4:1000" {
			if w.Code != 403 {
				t.Fatal("TUV must stay behind ingress")
			}
			continue
		}
		if w.Code != 200 || strings.Contains(w.Body.String(), "test-secret") || strings.Contains(w.Body.String(), "sensor.tepla_voda") {
			t.Fatal("TUV response fails or leaks configuration")
		}
		var out homeassistant.Snapshot
		if json.Unmarshal(w.Body.Bytes(), &out) != nil || out.Connected || out.NominalPower.Value != nil {
			t.Fatal("startup must not confirm old state")
		}
	}
}

func TestPumpTelemetryCannotActivateControl(t *testing.T) {
	a := &App{ha: homeassistant.New(homeassistant.DefaultConfig(), "test-secret"), pump: pump.New(pump.Config{Enabled: true, FreshSeconds: 120}, nil, "", nil)}
	h := a.handler(t.TempDir(), true)
	r := httptest.NewRequest("GET", "/api/tuv", nil)
	r.RemoteAddr = "172.30.32.2:1000"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var out struct {
		Pump pump.Snapshot `json:"pump_control"`
	}
	if json.Unmarshal(w.Body.Bytes(), &out) != nil || !out.Pump.Enabled || out.Pump.Status != "waiting" || out.Pump.Confirmed != nil {
		t.Fatal("pump ownership must not imply an initialized output")
	}
	if strings.Contains(w.Body.String(), "test-secret") || strings.Contains(w.Body.String(), "switch.kicony") {
		t.Fatal("controller API must not expose configuration")
	}
	for _, path := range []string{"/api/pump", "/api/pump/enable", "/api/control", "/api/tuv"} {
		r = httptest.NewRequest("POST", path, strings.NewReader(`{"enabled":true,"desired":true}`))
		r.RemoteAddr = "172.30.32.2:1000"
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code < 400 {
			t.Fatal("dashboard must not offer manual pump control", path)
		}
	}
}

func TestTeslaAPIIngressAndPrivacy(t *testing.T) {
	a := &App{ha: homeassistant.New(homeassistant.DefaultConfig(), "test-secret")}
	h := a.handler(t.TempDir(), true)
	for _, peer := range []string{"1.2.3.4:1000", "172.30.32.2:1000"} {
		r := httptest.NewRequest("GET", "/api/tesla", nil)
		r.RemoteAddr = peer
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if peer == "1.2.3.4:1000" {
			if w.Code != 403 {
				t.Fatal("Tesla ingress")
			}
			continue
		}
		if w.Code != 200 || strings.Contains(w.Body.String(), "test-secret") || strings.Contains(w.Body.String(), "sensor.uroven_baterie") {
			t.Fatal("Tesla API privacy")
		}
		var out homeassistant.TeslaSnapshot
		if json.Unmarshal(w.Body.Bytes(), &out) != nil || !out.Enabled || out.Connected || out.Readings["soc"].Value != nil {
			t.Fatal("startup must wait for telemetry")
		}
	}
}

func TestVictronAPIIngressAndPrivacy(t *testing.T) {
	a := &App{victron: victron.New(victron.Config{Password: "test-secret"})}
	h := a.handler(t.TempDir(), true)
	for _, peer := range []string{"1.2.3.4:1000", "172.30.32.2:1000"} {
		r := httptest.NewRequest("GET", "/api/victron", nil)
		r.RemoteAddr = peer
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if peer == "1.2.3.4:1000" {
			if w.Code != 403 {
				t.Fatal("telemetry must stay behind ingress")
			}
			continue
		}
		if w.Code != 200 || strings.Contains(w.Body.String(), "test-secret") {
			t.Fatal("telemetry response leaks credentials or fails")
		}
		var snapshot victron.Snapshot
		if err := json.Unmarshal(w.Body.Bytes(), &snapshot); err != nil || snapshot.Enabled || snapshot.Connected {
			t.Fatal("default must remain disabled")
		}
	}
}

func TestAPIAndPersistence(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec("CREATE TABLE settings(key TEXT PRIMARY KEY,value TEXT); CREATE TABLE history(at INTEGER,battery REAL,soc REAL,min_cell REAL,max_cell REAL,tuv REAL,car REAL,reason TEXT)")
	if err != nil {
		t.Fatal(err)
	}
	a := &App{db: db, scenario: "sunny"}
	if err = a.tick(time.Now()); err != nil {
		t.Fatal(err)
	}
	h := a.handler(t.TempDir(), true)
	r := httptest.NewRequest("GET", "/api/state", nil)
	r.RemoteAddr = "1.2.3.4:1000"
	r.Header.Set("X-Forwarded-For", "172.30.32.2")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("must reject spoofed ingress")
	}
	r.RemoteAddr = "172.30.32.2:1000"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var state map[string]any
	if err = json.Unmarshal(w.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if state["observe_only"] != true || state["control_enabled"] != false {
		t.Fatal("unsafe mode")
	}
	r = httptest.NewRequest("POST", "/api/scenario", strings.NewReader(`{"name":"critical"}`))
	r.RemoteAddr = "172.30.32.2:1000"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatal(w.Body.String())
	}
	var count int
	if err = db.QueryRow("SELECT count(*) FROM history").Scan(&count); err != nil || count != 2 {
		t.Fatalf("persistence %d %v", count, err)
	}
	r = httptest.NewRequest("POST", "/api/control", nil)
	r.RemoteAddr = "172.30.32.2:1000"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code == 200 {
		t.Fatal("control must not exist")
	}
}

func TestFixedPumpControlAPI(t *testing.T) {
	p := pump.New(pump.Config{Enabled: true, FreshSeconds: 120}, nil, "", nil)
	a := &App{pump: p, tuv: tuv.New(p, "auto", func(string) error { return nil })}
	h := a.handler(t.TempDir(), true)
	call := func(method, path, body, token, peer string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.RemoteAddr = peer
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-FVE-TUV", token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	peer := "172.30.32.2:1000"
	w := call("GET", "/api/tuv", "", "", peer)
	var out struct {
		Token string `json:"control_token"`
	}
	if json.Unmarshal(w.Body.Bytes(), &out) != nil || out.Token == "" {
		t.Fatal("missing request token")
	}
	for _, tc := range []struct {
		path, body, token, peer string
		code                    int
	}{
		{"/api/tuv/mode", `{"mode":"manual"}`, "", peer, 403},
		{"/api/tuv/mode", `{"mode":"manual"}`, out.Token, "1.2.3.4:1", 403},
		{"/api/tuv/pump", `{"on":true}`, out.Token, peer, 409},
		{"/api/tuv/mode", `{"mode":"manual","entity_id":"switch.other"}`, out.Token, peer, 400},
		{"/api/tuv/mode", `{"mode":"manual"} {}`, out.Token, peer, 400},
		{"/api/tuv/mode", `{"mode":"manual"}`, out.Token, peer, 202},
		{"/api/tuv/pump", `{}`, out.Token, peer, 400},
		{"/api/tuv/pump", `{"on":true}`, out.Token, peer, 409},
		{"/api/tuv/pump", `{"on":false}`, out.Token, peer, 202},
		{"/api/tuv/power", `{"watts":3000}`, out.Token, peer, 404},
		{"/api/services", `{"service":"turn_on"}`, out.Token, peer, 404},
	} {
		w = call("POST", tc.path, tc.body, tc.token, tc.peer)
		if w.Code != tc.code {
			t.Fatalf("%s %s: got %d want %d (%s)", tc.path, tc.body, w.Code, tc.code, w.Body.String())
		}
	}
}
