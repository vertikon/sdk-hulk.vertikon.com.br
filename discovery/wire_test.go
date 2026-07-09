package discovery

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Prova offline (sem broker) que os payloads casam com o wire esperado pelo
// mcp-inventory (pkg/contracts): nomes de campo snake_case exatos.
func TestWireContract(t *testing.T) {
	reg, _ := json.Marshal(registerPayload{Slug: "ia", Version: "1.0.0", Environment: "prod", Host: "VM1", Kind: "mcp", SDKVersion: "0.1.0"})
	for _, k := range []string{`"slug":"ia"`, `"version":"1.0.0"`, `"environment":"prod"`, `"host":"VM1"`, `"kind":"mcp"`, `"sdk_version":"0.1.0"`} {
		if !strings.Contains(string(reg), k) {
			t.Errorf("register sem campo %s: %s", k, reg)
		}
	}
	hb, _ := json.Marshal(heartbeatPayload{Slug: "ia", Environment: "prod", Host: "VM1", Version: "1.0.0", StatusReported: "ok", UptimeSeconds: 42, Checks: map[string]bool{"db": true}, SentAt: time.Unix(1_700_000_000, 0).UTC()})
	for _, k := range []string{`"slug":"ia"`, `"status_reported":"ok"`, `"uptime_seconds":42`, `"checks":{"db":true}`, `"sent_at":`} {
		if !strings.Contains(string(hb), k) {
			t.Errorf("heartbeat sem campo %s: %s", k, hb)
		}
	}
}
