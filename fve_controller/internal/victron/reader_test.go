package victron

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testConfig() Config {
	return Config{Enabled: true, Host: "127.0.0.1", Port: 1883, FreshSeconds: 60, Topics: map[string]string{"min_cell": "victron/N/test/battery/0/System/MinCellVoltage", "battery": "victron/N/test/battery/0/Dc/0/Power"}}
}

func TestTelemetryQuality(t *testing.T) {
	now := time.Now()
	r := New(testConfig())
	r.connected = true
	topic := r.config.Topics["battery"]
	if r.Snapshot(now).Readings["battery"].Quality != "missing" {
		t.Fatal("missing must not be zero")
	}
	r.receive(topic, []byte(`{"value":0}`), false, now)
	s := r.Snapshot(now).Readings["battery"]
	if s.Quality != "valid" || s.Value == nil || *s.Value != 0 {
		t.Fatal("zero is a valid measurement")
	}
	if r.Snapshot(now.Add(61 * time.Second)).Readings["battery"].Quality != "stale" {
		t.Fatal("stale must not be refreshed by snapshot")
	}
	for _, v := range []float64{-700, 700} {
		r.receive(topic, []byte(fmt.Sprintf(`{"value":%g}`, v)), false, now)
		if *r.Snapshot(now).Readings["battery"].Value != v {
			t.Fatal("battery sign must be preserved until verified")
		}
	}
	for _, payload := range []string{"", `{"value":null}`, `{}`, `{"value":"0"}`, `{"value":1e999}`, `broken`, `{"value":2} trailing`} {
		r.receive(topic, []byte(payload), false, now)
		s = r.Snapshot(now).Readings["battery"]
		if s.Quality != "invalid" || s.Value != nil {
			t.Fatalf("invalid payload %q", payload)
		}
	}
	r.receive(topic, []byte(`{"value":800}`), true, now)
	if r.Snapshot(now).Readings["battery"].Quality != "retained" {
		t.Fatal("retained has unknown age")
	}
	r.receive(topic, []byte(`{"value":800}`), false, now)
	r.connected = false
	if r.Snapshot(now).Readings["battery"].Quality != "offline" {
		t.Fatal("offline cannot confirm a measurement")
	}
	r.reset()
	r.connected = true
	if r.Snapshot(now).Readings["battery"].Value != nil {
		t.Fatal("reconnect cannot restore old confirmed data")
	}
	r.receive(r.config.Topics["min_cell"], []byte(`{"value":3.274}`), false, now)
	if r.Snapshot(now).Readings["min_cell"].Quality != "valid" {
		t.Fatal("cell voltage")
	}
	r.receive(r.config.Topics["min_cell"], []byte(`{"value":0}`), false, now)
	if r.Snapshot(now).Readings["min_cell"].Quality != "invalid" {
		t.Fatal("zero cell voltage cannot be used")
	}
}

func TestMappedMPPTAndAggregatedBattery(t *testing.T) {
	c := testConfig()
	c.Topics = map[string]string{
		"solar_shelter": "victron/N/test/solarcharger/278/Yield/Power",
		"solar_fence":   "victron/N/test/solarcharger/1/Yield/Power",
		"solar_roof":    "victron/N/test/solarcharger/0/Yield/Power",
		"min_cell":      "victron/N/test/battery/99/System/MinCellVoltage",
		"soc":           "victron/N/test/battery/278/Soc",
		"battery":       "victron/N/test/battery/278/Dc/0/Power",
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	r := New(c)
	r.connected = true
	now := time.Now()
	for key, value := range map[string]float64{
		"solar_shelter": 590.25,
		"solar_fence":   4.820000171661377,
		"solar_roof":    105.37000274658203,
		"min_cell":      3.435,
		// SOC is synthetic; the negative power sample was reported during discharge.
		"soc":     93,
		"battery": -170.7480010986328,
	} {
		r.receive(c.Topics[key], []byte(fmt.Sprintf(`{"value":%.17g}`, value)), false, now)
		s := r.Snapshot(now).Readings[key]
		if s.Quality != "valid" || s.Value == nil || *s.Value != value || s.Unit != units[key] {
			t.Fatalf("incorrect mapping or precision for %s: %+v", key, s)
		}
	}
	// An old battery instance cannot replace the configured aggregate reading.
	r.receive("victron/N/test/battery/0/System/MinCellVoltage", []byte(`{"value":3.2}`), false, now)
	if got := r.Snapshot(now).Readings["min_cell"].Value; got == nil || *got != 3.435 {
		t.Fatal("unconfigured battery instance replaced the aggregate")
	}
	for _, key := range []string{"max_cell"} {
		if r.Snapshot(now).Readings[key].Quality != "not_configured" {
			t.Fatal("unknown topics must not become simulated readings")
		}
	}
}

func TestOptionsSafety(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.json")
	c, err := Load(path)
	if err != nil || c.Enabled {
		t.Fatal("missing options defaults to disabled")
	}
	for _, topic := range []string{"victron/W/test/battery/0/Dc/0/Power", "victron/R/test/keepalive", "victron/N/test/#"} {
		c = testConfig()
		c.Topics["battery"] = topic
		if c.Validate() == nil {
			t.Fatalf("unsafe subscription %s", topic)
		}
	}
	c = testConfig()
	c.Password = "test-secret"
	c.Topics["battery"] = "bad"
	if err = c.Validate(); err == nil || strings.Contains(err.Error(), c.Password) {
		t.Fatal("validation must redact secrets")
	}
	if err = os.WriteFile(path, []byte(`{"mqtt_enabled":false,"mqtt_password":"test-secret"}`), 0600); err != nil {
		t.Fatal(err)
	}
	c, err = Load(path)
	if err != nil || c.Password != "test-secret" {
		t.Fatal("read backend-only options")
	}
	b, _ := json.Marshal(New(c).Snapshot(time.Now()))
	if strings.Contains(string(b), "test-secret") {
		t.Fatal("API leaked credentials")
	}
}

func TestInverterTopicUpgradeAndQuality(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.json")
	if err := os.WriteFile(path, []byte(`{"mqtt_enabled":true,"mqtt_topics":{"battery":"victron/N/test/battery/0/Dc/0/Power"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil || c.Topics["inverter"] != defaultInverterTopic || c.Topics["battery"] != "victron/N/test/battery/0/Dc/0/Power" {
		t.Fatal("upgrade must add inverter and preserve saved topics")
	}
	for _, topic := range []string{"", "victron/N/test/system/0/Ac/ConsumptionOnOutput/L1/Power"} {
		if err := os.WriteFile(path, []byte(fmt.Sprintf(`{"mqtt_topics":{"inverter":%q}}`, topic)), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil || got.Topics["inverter"] != topic {
			t.Fatal("explicit setting must be preserved")
		}
	}
	c.Topics["inverter"] = "victron/N/test/system/0/Ac/ConsumptionOnOutput/L1/Power"
	r := New(c)
	r.connected = true
	now := time.Now()
	topic := c.Topics["inverter"]
	for _, value := range []float64{0, 425.5, 8200} {
		r.receive(topic, []byte(fmt.Sprintf(`{"value":%g}`, value)), false, now)
		s := r.Snapshot(now).Readings["inverter"]
		if s.Quality != "valid" || s.Value == nil || *s.Value != value || s.Unit != "W" {
			t.Fatal("inverter zero, precision or overrange")
		}
	}
	if r.Snapshot(now.Add(61 * time.Second)).Readings["inverter"].Quality != "stale" {
		t.Fatal("stale inverter")
	}
	for _, payload := range []string{`{"value":-1}`, `{"value":null}`, `{"value":"425"}`, `{"value":1e999}`} {
		r.receive(topic, []byte(payload), false, now)
		s := r.Snapshot(now).Readings["inverter"]
		if s.Quality != "invalid" || s.Value != nil {
			t.Fatal("invalid inverter")
		}
	}
	r.receive(topic, []byte(`{"value":425}`), true, now)
	if r.Snapshot(now).Readings["inverter"].Quality != "retained" {
		t.Fatal("retained is not current")
	}
	r.receive(topic, []byte(`{"value":425}`), false, now)
	r.connected = false
	if r.Snapshot(now).Readings["inverter"].Quality != "offline" {
		t.Fatal("outage")
	}
	r.reset()
	if r.Snapshot(now).Readings["inverter"].Value != nil {
		t.Fatal("restart must wait for a new measurement")
	}
}

func TestSubscriptionDenied(t *testing.T) {
	topics := map[string]byte{"victron/N/test/battery/0/Soc": 0}
	for _, results := range []map[string]byte{{}, {"victron/N/test/battery/0/Soc": 0x80}} {
		if subscriptionAccepted(results, topics) {
			t.Fatal("missing or denied SUBACK must not report listening")
		}
	}
}

// Tiny local broker fixture checks the actual wire path: CONNECT, SUBSCRIBE,
// receive live data, then disconnect. No external service or private fixture.
func TestMQTTWireReadOnly(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	done := make(chan error, 1)
	go func() {
		c, err := l.Accept()
		if err != nil {
			done <- err
			return
		}
		defer c.Close()
		c.SetDeadline(time.Now().Add(5 * time.Second))
		read := func() (byte, []byte, error) {
			h := make([]byte, 1)
			if _, err := io.ReadFull(c, h); err != nil {
				return 0, nil, err
			}
			size, mult := 0, 1
			for i := 0; i < 4; i++ {
				b := make([]byte, 1)
				if _, err := io.ReadFull(c, b); err != nil {
					return 0, nil, err
				}
				size += int(b[0]&127) * mult
				if b[0]&128 == 0 {
					body := make([]byte, size)
					_, err := io.ReadFull(c, body)
					return h[0], body, err
				}
				mult *= 128
			}
			return 0, nil, fmt.Errorf("packet too long")
		}
		h, _, err := read()
		if err != nil || h>>4 != 1 {
			done <- fmt.Errorf("expected CONNECT: %v", err)
			return
		}
		c.Write([]byte{0x20, 2, 0, 0})
		h, b, err := read()
		if err != nil || h>>4 != 8 || len(b) < 2 {
			done <- fmt.Errorf("expected SUBSCRIBE: %v", err)
			return
		}
		if !strings.Contains(string(b), "victron/N/test/") || strings.Contains(string(b), "/W/") {
			done <- fmt.Errorf("unexpected topics")
			return
		}
		c.Write([]byte{0x90, 4, b[0], b[1], 0, 0})
		topic := "victron/N/test/battery/0/Dc/0/Power"
		body := append([]byte{0, byte(len(topic))}, []byte(topic)...)
		body = append(body, []byte(`{"value":-321}`)...)
		c.Write(append([]byte{0x30, byte(len(body))}, body...))
		h, _, err = read()
		if err != nil {
			done <- err
			return
		}
		if h>>4 != 14 {
			done <- fmt.Errorf("unexpected outbound packet %x", h)
			return
		}
		done <- nil
	}()
	c := testConfig()
	c.Port = l.Addr().(*net.TCPAddr).Port
	r := New(c)
	r.Start()
	defer r.Close()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s := r.Snapshot(time.Now())
		v := s.Readings["battery"]
		if s.Status == "listening" && v.Quality == "valid" && v.Value != nil && *v.Value == -321 {
			r.Close()
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("live MQTT data not received")
}
