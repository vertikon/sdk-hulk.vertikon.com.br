// Exemplo: como qualquer MCP adere ao discovery do ecossistema em 3 passos.
// Sobe → registra + heartbeat automáticos → aparece no mcp-inventory sem cadastro.
//
//	NATS_URL=nats://localhost:4222 go run ./examples/discovery-piloto
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vertikon/sdk-hulk.vertikon.com.br/discovery"
)

func main() {
	url := os.Getenv("NATS_URL")
	if url == "" {
		url = "nats://localhost:4222"
	}
	// 1) configurar (slug bate com o catálogo do inventário)
	c, err := discovery.Connect(url, discovery.Config{
		Slug: "ia-piloto", Name: "IA (piloto)", Environment: "prod", Host: "VM1",
		Version: "1.0.0", SDKVersion: "0.1.0", Interval: 5 * time.Second,
		Checks: map[string]discovery.Check{
			"nats": func(context.Context) bool { return true },
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	// 2) iniciar (register + heartbeat loop)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := c.Start(ctx); err != nil {
		log.Fatal(err)
	}
	log.Println("piloto no ar — heartbeat a cada 5s; Ctrl-C para sair (deregister gracioso)")
	<-ctx.Done()
	// 3) parar (deregister gracioso)
	c.Stop()
}
