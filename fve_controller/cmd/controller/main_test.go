package main

import (
	"database/sql"
	"encoding/json"
	"fve-controller/internal/victron"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
