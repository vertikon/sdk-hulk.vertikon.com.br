// Package discovery é o client de descoberta do ecossistema Vertikon: qualquer
// MCP importa e ganha auto-registro + heartbeat + deregister "de graça",
// aparecendo no mcp-inventory (inventario.vertikon.com.br) sem cadastro manual.
//
// Contrato canônico (wire): `pkg/contracts` do mcp-inventory. Aqui as structs
// espelham esse formato JSON — publicamos por core NATS; a stream JetStream
// INVENTORY_INGEST do inventário captura os subjects. Sem dependência do módulo
// do inventário (evita ciclo e deps pesadas no SDK).
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// Subjects do protocolo v1 (§5.1 do blueprint do inventário).
const (
	subjectRegister   = "vertikon.inventory.v1.register"
	subjectDeregister = "vertikon.inventory.v1.deregister"
	subjectHeartbeat  = "vertikon.inventory.v1.heartbeat.%s" // + slug
)

// Check é uma verificação plugável de dependência (ex.: db, nats). true = ok.
type Check func(ctx context.Context) bool

// Config do client de discovery.
type Config struct {
	Slug        string            // obrigatório (kebab-case, bate com o catálogo)
	Name        string            // opcional (usado se o sistema ainda não existir)
	Environment string            // dev|staging|prod
	Host        string            // VM/hostname
	Version     string            // semver do binário
	Kind        string            // mcp|agent|... (default mcp)
	SDKVersion  string            // versão do sdk-hulk (rastreio de rollout)
	Interval    time.Duration     // intervalo do heartbeat (default 30s)
	Checks      map[string]Check  // dependências reportadas no heartbeat
}

// Client publica registro e heartbeats para o inventário.
type Client struct {
	nc     *nats.Conn
	cfg    Config
	start  time.Time
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type registerPayload struct {
	Slug        string `json:"slug"`
	Name        string `json:"name,omitempty"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
	Host        string `json:"host"`
	Kind        string `json:"kind"`
	SDKVersion  string `json:"sdk_version"`
}

type heartbeatPayload struct {
	Slug           string          `json:"slug"`
	Environment    string          `json:"environment"`
	Host           string          `json:"host"`
	Version        string          `json:"version"`
	StatusReported string          `json:"status_reported"`
	UptimeSeconds  int64           `json:"uptime_seconds"`
	Checks         map[string]bool `json:"checks,omitempty"`
	SentAt         time.Time       `json:"sent_at"`
}

type deregisterPayload struct {
	Slug        string `json:"slug"`
	Environment string `json:"environment"`
	Host        string `json:"host"`
}

// Connect abre a conexão e prepara o client (aplica defaults).
func Connect(natsURL string, cfg Config) (*Client, error) {
	if cfg.Slug == "" || cfg.Environment == "" || cfg.Host == "" {
		return nil, fmt.Errorf("discovery: Slug, Environment e Host são obrigatórios")
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 30 * time.Second
	}
	if cfg.Kind == "" {
		cfg.Kind = "mcp"
	}
	nc, err := nats.Connect(natsURL, nats.Name("discovery/"+cfg.Slug))
	if err != nil {
		return nil, err
	}
	return &Client{nc: nc, cfg: cfg, start: time.Now()}, nil
}

// Start publica o registro e começa o loop de heartbeat até o Stop/ctx.
func (c *Client) Start(ctx context.Context) error {
	reg, _ := json.Marshal(registerPayload{
		Slug: c.cfg.Slug, Name: c.cfg.Name, Version: c.cfg.Version, Environment: c.cfg.Environment,
		Host: c.cfg.Host, Kind: c.cfg.Kind, SDKVersion: c.cfg.SDKVersion,
	})
	if err := c.nc.Publish(subjectRegister, reg); err != nil {
		return err
	}
	c.beat(ctx) // primeiro heartbeat imediato

	loopCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		t := time.NewTicker(c.cfg.Interval)
		defer t.Stop()
		for {
			select {
			case <-loopCtx.Done():
				return
			case <-t.C:
				c.beat(loopCtx)
			}
		}
	}()
	return nil
}

func (c *Client) beat(ctx context.Context) {
	checks := map[string]bool{}
	allOK := true
	for name, fn := range c.cfg.Checks {
		ok := fn(ctx)
		checks[name] = ok
		allOK = allOK && ok
	}
	status := "ok"
	if !allOK {
		status = "degraded"
	}
	hb, _ := json.Marshal(heartbeatPayload{
		Slug: c.cfg.Slug, Environment: c.cfg.Environment, Host: c.cfg.Host, Version: c.cfg.Version,
		StatusReported: status, UptimeSeconds: int64(time.Since(c.start).Seconds()),
		Checks: checks, SentAt: time.Now().UTC(),
	})
	_ = c.nc.Publish(fmt.Sprintf(subjectHeartbeat, c.cfg.Slug), hb)
	_ = c.nc.Flush()
}

// Stop publica o deregister (shutdown gracioso) e fecha a conexão.
func (c *Client) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
	dreg, _ := json.Marshal(deregisterPayload{Slug: c.cfg.Slug, Environment: c.cfg.Environment, Host: c.cfg.Host})
	_ = c.nc.Publish(subjectDeregister, dreg)
	_ = c.nc.Flush()
	c.nc.Close()
}
