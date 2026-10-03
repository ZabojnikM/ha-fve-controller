// Package victron receives telemetry only. It has no publish or control path.
package victron

import (
	"encoding/json"
	"errors"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Config struct {
	Enabled      bool              `json:"mqtt_enabled"`
	Host         string            `json:"mqtt_host"`
	Port         int               `json:"mqtt_port"`
	Username     string            `json:"mqtt_username"`
	Password     string            `json:"mqtt_password"`
	FreshSeconds int               `json:"mqtt_fresh_seconds"`
	Topics       map[string]string `json:"mqtt_topics"`
}

var units = map[string]string{"min_cell": "V", "max_cell": "V", "soc": "%", "battery": "W", "solar_roof": "W", "solar_shelter": "W", "solar_fence": "W", "inverter": "W"}

const defaultInverterTopic = "victron/N/c0619ab221ee/system/0/Ac/ConsumptionOnOutput/L1/Power"

func Load(path string) (Config, error) {
	// Preserve saved topics; older options without this key gain the confirmed topic.
	// An explicitly empty inverter topic stays disabled.
	c := Config{Host: "core-mosquitto", Port: 1883, FreshSeconds: 60, Topics: map[string]string{"inverter": defaultInverterTopic}}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, errors.New("nelze číst nastavení MQTT")
	}
	if json.Unmarshal(b, &c) != nil {
		return c, errors.New("neplatný formát nastavení MQTT")
	}
	return c, c.Validate()
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.Host) == "" || strings.ContainsAny(c.Host, "/@?#\x00\r\n ") || c.Port < 1 || c.Port > 65535 || c.FreshSeconds < 5 || c.FreshSeconds > 3600 {
		return errors.New("neplatný broker nebo stáří dat MQTT")
	}
	count := 0
	for key, topic := range c.Topics {
		if _, ok := units[key]; !ok {
			return errors.New("neznámé měření MQTT")
		}
		if topic == "" {
			continue
		}
		// Only exact Victron notification topics are accepted; no command topics.
		if !strings.Contains(topic, "/N/") && !strings.HasPrefix(topic, "N/") || strings.ContainsAny(topic, "+#\x00\r\n") || len(topic) > 65535 {
			return errors.New("očekáván přesný notifikační topic MQTT")
		}
		count++
	}
	if count == 0 {
		return errors.New("nastavte alespoň jeden topic MQTT")
	}
	return nil
}

type Reading struct {
	Value    *float64   `json:"value"`
	Unit     string     `json:"unit"`
	At       *time.Time `json:"at"`
	Quality  string     `json:"quality"`
	Retained bool       `json:"retained"`
}

type Snapshot struct {
	Enabled      bool               `json:"enabled"`
	Connected    bool               `json:"connected"`
	Status       string             `json:"status"`
	FreshSeconds int                `json:"fresh_seconds"`
	Readings     map[string]Reading `json:"readings"`
}

type Reader struct {
	mu        sync.Mutex
	config    Config
	connected bool
	status    string
	readings  map[string]Reading
	client    mqtt.Client
}

func New(c Config) *Reader {
	r := &Reader{config: c, status: "disabled"}
	r.reset()
	if c.Enabled {
		r.status = "connecting"
	}
	return r
}

func (r *Reader) reset() {
	r.readings = map[string]Reading{}
	for key, unit := range units {
		quality := "not_configured"
		if r.config.Topics[key] != "" {
			quality = "missing"
		}
		r.readings[key] = Reading{Unit: unit, Quality: quality}
	}
}

func (r *Reader) receive(topic string, payload []byte, retained bool, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, configured := range r.config.Topics {
		if configured == "" || topic != configured {
			continue
		}
		s := Reading{Unit: units[key], At: &now, Quality: "invalid", Retained: retained}
		var body struct {
			Value *float64 `json:"value"`
		}
		if len(payload) <= 4096 && json.Unmarshal(payload, &body) == nil && body.Value != nil && !math.IsNaN(*body.Value) && !math.IsInf(*body.Value, 0) {
			v := *body.Value
			valid := key != "soc" || v >= 0 && v <= 100
			if key == "min_cell" || key == "max_cell" {
				valid = v > 0 && v <= 5
			}
			if strings.HasPrefix(key, "solar_") || key == "inverter" {
				valid = v >= 0
			}
			if valid {
				s.Value = &v
				s.Quality = "valid"
				if retained {
					s.Quality = "retained"
				}
			}
		}
		r.readings[key] = s
	}
}

func (r *Reader) Snapshot(now time.Time) Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := Snapshot{Enabled: r.config.Enabled, Connected: r.connected, Status: r.status, FreshSeconds: r.config.FreshSeconds, Readings: map[string]Reading{}}
	for key, s := range r.readings {
		if s.Quality == "valid" {
			if !r.connected {
				s.Quality = "offline"
			} else if s.At == nil || now.Sub(*s.At) > time.Duration(r.config.FreshSeconds)*time.Second || s.At.After(now) {
				s.Quality = "stale"
			}
		}
		out.Readings[key] = s
	}
	return out
}

func (r *Reader) Start() {
	if !r.config.Enabled {
		return
	}
	o := mqtt.NewClientOptions().AddBroker("tcp://" + net.JoinHostPort(r.config.Host, strconv.Itoa(r.config.Port)))
	o.SetClientID("fve-observe-" + strconv.FormatInt(time.Now().UnixNano(), 36))
	o.SetUsername(r.config.Username).SetPassword(r.config.Password)
	o.SetCleanSession(true).SetAutoReconnect(true).SetConnectRetry(true).SetConnectRetryInterval(5 * time.Second)
	o.SetConnectTimeout(5 * time.Second).SetKeepAlive(30 * time.Second).SetMaxReconnectInterval(30 * time.Second)
	o.SetConnectionLostHandler(func(_ mqtt.Client, _ error) {
		r.mu.Lock()
		r.connected = false
		r.status = "offline"
		r.reset()
		r.mu.Unlock()
	})
	o.SetOnConnectHandler(func(c mqtt.Client) {
		r.mu.Lock()
		r.connected = true
		r.status = "subscribing"
		r.reset()
		r.mu.Unlock()
		topics := map[string]byte{}
		for _, t := range r.config.Topics {
			if t != "" {
				topics[t] = 0
			}
		}
		token := c.SubscribeMultiple(topics, func(_ mqtt.Client, m mqtt.Message) { r.receive(m.Topic(), m.Payload(), m.Retained(), time.Now()) })
		ok := token.WaitTimeout(5*time.Second) && token.Error() == nil
		if ok {
			subscription, typed := token.(*mqtt.SubscribeToken)
			ok = typed && subscriptionAccepted(subscription.Result(), topics)
		}
		r.mu.Lock()
		if ok && r.connected {
			r.status = "listening"
		} else if r.connected {
			r.status = "subscription_error"
			r.connected = false
			r.reset()
		}
		r.mu.Unlock()
		if !ok {
			c.Disconnect(0)
		}
	})
	r.client = mqtt.NewClient(o)
	// Retry happens inside Paho. Never log library errors or connection options:
	// they may contain credentials. MQTT does not block the web or simulator.
	r.client.Connect()
}

func subscriptionAccepted(results map[string]byte, topics map[string]byte) bool {
	for topic := range topics {
		qos, present := results[topic]
		if !present || qos > 2 {
			return false
		}
	}
	return true
}

func (r *Reader) Close() {
	if r.client != nil {
		r.client.Disconnect(100)
	}
}
