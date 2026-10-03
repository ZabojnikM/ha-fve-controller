package homeassistant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPumpTransportOnlyAddressesItsSwitchAndNeverConfirmsState(t *testing.T) {
	code := 200
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "POST" || (r.URL.Path != "/services/switch/turn_on" && r.URL.Path != "/services/switch/turn_off") {
			t.Error("unexpected command")
		}
		var payload map[string]string
		if json.NewDecoder(r.Body).Decode(&payload) != nil || len(payload) != 1 || payload["entity_id"] != "switch.test_pump" || r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Error("unexpected target or auth")
		}
		w.WriteHeader(code)
		w.Write([]byte("private-response"))
	}))
	defer server.Close()
	p := NewPumpCommands("switch.test_pump", "test-secret")
	p.baseURL = server.URL + "/services/switch/"
	for _, on := range []bool{true, false} {
		if err := p.Set(context.Background(), on); err != nil {
			t.Fatal(err)
		}
	}
	code = 403
	err := p.Set(context.Background(), false)
	if err == nil || strings.Contains(err.Error(), "test-secret") || strings.Contains(err.Error(), "private-response") {
		t.Fatal("command error must be sanitized")
	}
	p.entity = "select.test_heating"
	if p.Set(context.Background(), true) == nil || calls != 3 {
		t.Fatal("transport must reject other domains before network")
	}
	p.entity = "switch.test_pump"
	p.token = ""
	if p.Set(context.Background(), true) == nil || calls != 3 {
		t.Fatal("missing token must not call HA")
	}
}
