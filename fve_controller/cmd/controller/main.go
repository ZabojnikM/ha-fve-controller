package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fve-controller/internal/core"
	"fve-controller/internal/homeassistant"
	"fve-controller/internal/victron"
	"log"
	_ "modernc.org/sqlite"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type App struct {
	mu       sync.Mutex
	db       *sql.DB
	engine   core.Engine
	scenario string
	state    map[string]any
	victron  *victron.Reader
	ha       *homeassistant.Reader
}

func env(k, v string) string {
	if s := os.Getenv(k); s != "" {
		return s
	}
	return v
}
func sample(v float64, u string, n time.Time) core.Sample {
	return core.Sample{Value: v, Unit: u, At: n, Valid: true, Source: "simulator"}
}
func (a *App) tick(now time.Time) error {
	in := core.Input{SOC: sample(97, "%", now), Battery: sample(2800, "W", now), Temperature: sample(48, "°C", now), MinCell: sample(3.44, "V", now), MaxCell: sample(3.48, "V", now), TUV: sample(1000, "W", now), Car: sample(0, "W", now), Connected: true, Tariff: "VT"}
	switch a.scenario {
	case "zero":
		in.Battery.Value = 0
		in.TUV.Value = 0
	case "critical":
		in.SOC.Value = 8
	case "hot":
		in.Temperature.Value = 67
	case "stale":
		in.Battery.At = now.Add(-time.Minute)
	case "invalid":
		in.Battery.Valid = false
	case "overload":
		in.Overload = true
	case "balance":
		in.BalanceRequest = true
	case "manual":
		in.Manual = true
		in.ManualWatts = 2000
	}
	in.SolarRoof = sample(2600, "W", now)
	in.SolarShelter = sample(1400, "W", now)
	in.SolarFence = sample(600, "W", now)
	switch a.scenario {
	case "zero":
		in.SolarRoof.Value, in.SolarShelter.Value, in.SolarFence.Value = 450, 250, 100
	case "night":
		in.SolarRoof.Value, in.SolarShelter.Value, in.SolarFence.Value = 0, 0, 0
		in.Battery.Value, in.TUV.Value, in.SOC.Value = -800, 0, 70
	case "stale":
		in.SolarFence.At = now.Add(-time.Minute)
	case "invalid":
		in.SolarShelter.Valid = false
	}
	in.SolarTotal = core.SolarSum(now, [3]core.Sample{in.SolarRoof, in.SolarShelter, in.SolarFence})
	d := a.engine.Step(now, in)
	if !d.LastBalance.IsZero() {
		if _, err := a.db.Exec("INSERT INTO settings(key,value) VALUES('simulation_last_balance',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", d.LastBalance.Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	quality := map[string]string{}
	for k, s := range map[string]core.Sample{"soc": in.SOC, "battery": in.Battery, "temperature": in.Temperature, "min_cell": in.MinCell, "max_cell": in.MaxCell, "tuv": in.TUV, "car": in.Car} {
		quality[k] = s.Quality(now)
	}
	for k, s := range map[string]core.Sample{"solar_roof": in.SolarRoof, "solar_shelter": in.SolarShelter, "solar_fence": in.SolarFence, "solar_total": in.SolarTotal} {
		quality[k] = s.Quality(now)
	}
	a.state = map[string]any{"observe_only": true, "control_enabled": false, "source": "simulator", "scenario": a.scenario, "at": now, "input": in, "quality": quality, "decision": d, "outputs": map[string]any{"tuv": map[string]any{"desired": d.TUV, "sent": nil, "confirmed": in.TUV.Value, "status": "observe_only", "owner": "none"}, "car": map[string]any{"desired": d.Car, "sent": nil, "confirmed": in.Car.Value, "status": "observe_only", "owner": "none"}}}
	// History stores only fresh valid samples, never disguises an outage as a measurement.
	complete := true
	for _, k := range []string{"soc", "battery", "temperature", "min_cell", "max_cell", "tuv", "car"} {
		if quality[k] != "valid" {
			complete = false
		}
	}
	var err error
	if complete {
		_, err = a.db.Exec("INSERT INTO history(at,battery,soc,min_cell,max_cell,tuv,car,reason) VALUES(?,?,?,?,?,?,?,?)", now.Unix(), in.Battery.Value, in.SOC.Value, in.MinCell.Value, in.MaxCell.Value, d.TUV, d.Car, d.Reason)
	}
	if err != nil {
		return err
	}
	_, err = a.db.Exec("DELETE FROM history WHERE at < ?", now.Add(-7*24*time.Hour).Unix())
	return err
}
func (a *App) handler(web string, ingress bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/tuv", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		reader := a.ha
		if reader == nil {
			reader = homeassistant.New(homeassistant.Config{}, "")
		}
		json.NewEncoder(w).Encode(reader.Snapshot(time.Now()))
	})
	mux.HandleFunc("GET /api/victron", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		reader := a.victron
		if reader == nil {
			reader = victron.New(victron.Config{})
		}
		json.NewEncoder(w).Encode(reader.Snapshot(time.Now()))
	})
	mux.HandleFunc("GET /api/state", func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(a.state)
	})
	mux.HandleFunc("POST /api/scenario", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name string `json:"name"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&body) != nil {
			http.Error(w, "Invalid request", 400)
			return
		}
		switch body.Name {
		case "sunny", "zero", "night", "critical", "hot", "stale", "invalid", "overload", "balance", "manual":
		default:
			http.Error(w, "Unknown scenario", 400)
			return
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		a.scenario = body.Name
		a.engine = core.Engine{LastBalance: a.engine.LastBalance}
		if err := a.tick(time.Now()); err != nil {
			http.Error(w, "Storage error", 500)
			return
		}
		w.WriteHeader(204)
	})
	mux.HandleFunc("GET /api/history", func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()
		rows, err := a.db.Query("SELECT at,battery,soc,min_cell,max_cell FROM history ORDER BY at DESC LIMIT 120")
		if err != nil {
			http.Error(w, "Storage error", 500)
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var at int64
			var b, s, mi, ma float64
			if rows.Scan(&at, &b, &s, &mi, &ma) != nil {
				http.Error(w, "Storage error", 500)
				return
			}
			out = append(out, map[string]any{"at": at, "battery": b, "soc": s, "min_cell": mi, "max_cell": ma})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if a.db.Ping() != nil {
			http.Error(w, "Storage unavailable", 503)
			return
		}
		w.Write([]byte("ok"))
	})
	mux.Handle("/", http.FileServer(http.Dir(web)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peer, _, _ := net.SplitHostPort(r.RemoteAddr)
		if ingress && peer != "172.30.32.2" {
			http.Error(w, "Ingress only", 403)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		mux.ServeHTTP(w, r)
	})
}
func main() {
	dir := env("DATA_DIR", "data")
	if err := os.MkdirAll(dir, 0700); err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "simulation.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err = db.Exec("PRAGMA journal_mode=WAL; CREATE TABLE IF NOT EXISTS settings(key TEXT PRIMARY KEY,value TEXT NOT NULL); CREATE TABLE IF NOT EXISTS history(at INTEGER,battery REAL,soc REAL,min_cell REAL,max_cell REAL,tuv REAL,car REAL,reason TEXT); CREATE INDEX IF NOT EXISTS history_at ON history(at);")
	if err != nil {
		log.Fatal(err)
	}
	a := &App{db: db, scenario: "sunny"}
	mqttConfig, err := victron.Load(env("OPTIONS_PATH", filepath.Join(dir, "options.json")))
	if err != nil {
		log.Fatal(err)
	}
	a.victron = victron.New(mqttConfig)
	a.victron.Start()
	defer a.victron.Close()
	haConfig, err := homeassistant.Load(env("OPTIONS_PATH", filepath.Join(dir, "options.json")))
	if err != nil {
		log.Fatal(err)
	}
	a.ha = homeassistant.New(haConfig, os.Getenv("SUPERVISOR_TOKEN"))
	haContext, cancelHA := context.WithCancel(context.Background())
	defer cancelHA()
	a.ha.Start(haContext)
	var last string
	err = db.QueryRow("SELECT value FROM settings WHERE key='simulation_last_balance'").Scan(&last)
	if err == nil {
		a.engine.LastBalance, err = time.Parse(time.RFC3339Nano, last)
		if err != nil {
			log.Fatal(err)
		}
	} else if err != sql.ErrNoRows {
		log.Fatal(err)
	}
	if err = a.tick(time.Now()); err != nil {
		log.Fatal(err)
	}
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for now := range ticker.C {
			a.mu.Lock()
			err := a.tick(now)
			a.mu.Unlock()
			if err != nil {
				log.Fatal("Simulation storage failed")
			}
		}
	}()
	log.Print("FVE simulator: observe_only=true control_enabled=false")
	server := &http.Server{Addr: env("LISTEN_ADDR", "127.0.0.1:8099"), Handler: a.handler(env("WEB_DIR", "web/dist"), os.Getenv("INGRESS_ONLY") == "true"), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	log.Fatal(server.ListenAndServe())
}
