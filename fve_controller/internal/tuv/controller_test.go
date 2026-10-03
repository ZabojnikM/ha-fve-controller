package tuv

import (
	"errors"
	"fve-controller/internal/pump"
	"testing"
	"time"
)

func TestModePersistenceAndUnownedOutput(t *testing.T) {
	p := pump.New(pump.Config{Enabled: true, FreshSeconds: 120}, nil, "", nil)
	fail := true
	saved := ""
	c := New(p, "auto", func(mode string) error {
		if fail {
			return errors.New("disk error")
		}
		saved = mode
		return nil
	})
	if c.SetMode("manual") == nil || c.Snapshot().Mode != "auto" || p.Snapshot().Mode != "auto" {
		t.Fatal("failed persistence must not switch mode")
	}
	fail = false
	if c.SetMode("manual") != nil || saved != "manual" || p.Snapshot().Mode != "manual" {
		t.Fatal("central mode must propagate")
	}
	if c.SetMode("invalid") == nil {
		t.Fatal("invalid mode")
	}
	restart := New(pump.New(pump.Config{Enabled: true, FreshSeconds: 120}, nil, "", nil), saved, nil)
	if restart.Snapshot().Mode != "manual" || restart.Snapshot().AutomaticHeating || restart.Snapshot().HeatingOwner != "node_red" {
		t.Fatal("restart mode must leave heating with Node-RED")
	}
	if restart.pump.Snapshot().Confirmed != nil || *restart.pump.Snapshot().ManualRequest {
		t.Fatal("manual startup must be OFF and unconfirmed")
	}
	unowned := New(pump.New(pump.Config{}, nil, "", nil), "auto", func(string) error { return nil })
	if unowned.SetMode("manual") == nil || unowned.ManualPump(true, time.Now()) == nil {
		t.Fatal("unowned pump cannot be controlled")
	}
}
