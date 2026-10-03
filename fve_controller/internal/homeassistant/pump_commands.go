package homeassistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

// PumpCommands is deliberately separate from Reader. It can only address the
// configured pump switch, never heating, heartbeat, reset, or other services.
type PumpCommands struct {
	entity, token, baseURL string
	client                 *http.Client
}

func NewPumpCommands(entity, token string) *PumpCommands {
	return &PumpCommands{entity: entity, token: token, baseURL: "http://supervisor/core/api/services/switch/",
		client: &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (p *PumpCommands) Set(ctx context.Context, on bool) error {
	if p.token == "" {
		return errors.New("Chybí přístup k HA")
	}
	if !entityPattern.MatchString(p.entity) || !strings.HasPrefix(p.entity, "switch.") {
		return errors.New("Neplatné mapování čerpadla")
	}
	service := "turn_off"
	if on {
		service = "turn_on"
	}
	body, _ := json.Marshal(map[string]string{"entity_id": p.entity})
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+service, bytes.NewReader(body))
	if err != nil {
		return errors.New("Povel čerpadla nelze připravit")
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return errors.New("Povel čerpadla se nepodařilo doručit")
	}
	defer resp.Body.Close()
	// Ignore returned HA states; a subsequent read must confirm the output.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024*1024))
	if resp.StatusCode != http.StatusOK {
		return errors.New("HA odmítl povel čerpadla")
	}
	return nil
}
