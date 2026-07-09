package discovery_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/vertikon/sdk-hulk.vertikon.com.br/discovery"
)

// Integração NATS: o client publica register + heartbeat no wire esperado pelo
// inventário. Pula sem INVENTORY_TEST_NATS_URL (ex.: nats://localhost:4222).
func TestClientPublishes(t *testing.T) {
	url := os.Getenv("INVENTORY_TEST_NATS_URL")
	if url == "" {
		t.Skip("INVENTORY_TEST_NATS_URL não setado — pulando")
	}
	sub, err := nats.Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()

	regCh := make(chan []byte, 1)
	hbCh := make(chan []byte, 1)
	_, _ = sub.Subscribe("vertikon.inventory.v1.register", func(m *nats.Msg) { select { case regCh <- m.Data: default: } })
	_, _ = sub.Subscribe("vertikon.inventory.v1.heartbeat.>", func(m *nats.Msg) { select { case hbCh <- m.Data: default: } })
	_ = sub.Flush()

	c, err := discovery.Connect(url, discovery.Config{
		Slug: "ia-piloto", Environment: "prod", Host: "VM1", Version: "1.0.0", SDKVersion: "0.1.0",
		Checks: map[string]discovery.Check{"db": func(context.Context) bool { return true }},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer c.Stop()

	reg := recv(t, regCh)
	if reg["slug"] != "ia-piloto" || reg["sdk_version"] != "0.1.0" || reg["kind"] != "mcp" {
		t.Fatalf("register inválido: %+v", reg)
	}
	hb := recv(t, hbCh)
	if hb["slug"] != "ia-piloto" || hb["status_reported"] != "ok" {
		t.Fatalf("heartbeat inválido: %+v", hb)
	}
	if checks, ok := hb["checks"].(map[string]any); !ok || checks["db"] != true {
		t.Fatalf("checks não reportados: %+v", hb["checks"])
	}
}

func recv(t *testing.T, ch chan []byte) map[string]any {
	t.Helper()
	select {
	case data := <-ch:
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatal(err)
		}
		return m
	case <-time.After(4 * time.Second):
		t.Fatal("nada recebido no prazo")
		return nil
	}
}
