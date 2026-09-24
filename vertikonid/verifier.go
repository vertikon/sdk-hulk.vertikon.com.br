// Package vertikonid valida os access tokens JWT RS256 do Vertikon ID
// (https://id.vertikon.com.br) via JWKS e expõe as claims da fundação
// (tenant_id/vertical/plan — DEVKIT do ID §3d). Regra do ecossistema: todo login
// é pelo Vertikon ID; este é o verificador canônico (só stdlib).
// Para EMITIR tokens use o próprio IdP.
package vertikonid

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

// DefaultIssuer é o Authorization Server de produção.
const DefaultIssuer = "https://id.vertikon.com.br"

// PlanSuspended é o valor de plan quando o billing suspendeu o tenant —
// todo gateway deve bloquear (contrato §3d do DEVKIT do ID).
const PlanSuspended = "suspended"

// Erros de validação.
var (
	ErrTokenMalformed   = errors.New("vertikonid: token malformado")
	ErrTokenExpired     = errors.New("vertikonid: token expirado ou sem exp")
	ErrSignatureInvalid = errors.New("vertikonid: assinatura inválida")
	ErrKeyNotFound      = errors.New("vertikonid: kid não encontrado no JWKS")
	ErrIssuerMismatch   = errors.New("vertikonid: issuer ausente ou não confere")
	ErrAlgUnsupported   = errors.New("vertikonid: alg não suportado (só RS256)")
	ErrAudienceMismatch = errors.New("vertikonid: token não foi emitido para esta API (aud)")
)

// Claims são as claims que a fundação honra em todo token do ID.
type Claims struct {
	Sub      string `json:"sub"`
	Iss      string `json:"iss"`
	TenantID string `json:"tenant_id"`
	Vertical string `json:"vertical"`
	Plan     string `json:"plan"`
	Scope    string `json:"scope"`
	Exp      int64  `json:"exp"`
	Iat      int64  `json:"iat"`
	// Aud: APIs para as quais o token foi emitido (claim aud, string ou lista).
	Aud []string `json:"-"`
}

// Suspended informa se o tenant está suspenso pelo billing.
func (c Claims) Suspended() bool { return c.Plan == PlanSuspended }

// HasScope verifica um escopo na claim space-separated.
func (c Claims) HasScope(s string) bool {
	for _, sc := range strings.Fields(c.Scope) {
		if sc == s {
			return true
		}
	}
	return false
}

// Verifier valida tokens contra o JWKS do issuer, com cache de chaves por kid
// (refresh sob demanda quando aparece kid desconhecido — rotação de chave).
type Verifier struct {
	Issuer string
	// Audience, se não vazio, é exigido no aud do token (contrato D2 do ID: a API
	// pina o aud além do iss — senão aceita token emitido para outro serviço).
	Audience string
	HTTP     *http.Client

	mu   sync.RWMutex
	keys map[string]*rsa.PublicKey
	last time.Time
}

// New cria um Verifier. issuer vazio usa DefaultIssuer.
func New(issuer string) *Verifier {
	if issuer == "" {
		issuer = DefaultIssuer
	}
	return &Verifier{
		Issuer: strings.TrimRight(issuer, "/"),
		HTTP:   &http.Client{Timeout: 10 * time.Second},
		keys:   map[string]*rsa.PublicKey{},
	}
}

// Verify valida assinatura, exp e issuer (ambos OBRIGATÓRIOS) e devolve as claims.
// NÃO bloqueia plan=suspended — o chamador decide via Claims.Suspended()
// (o Middleware bloqueia).
func (v *Verifier) Verify(ctx context.Context, token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrTokenMalformed
	}
	headB, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrTokenMalformed
	}
	var head struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headB, &head); err != nil {
		return nil, ErrTokenMalformed
	}
	if head.Alg != "RS256" {
		return nil, ErrAlgUnsupported
	}

	key, err := v.key(ctx, head.Kid)
	if err != nil {
		return nil, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrTokenMalformed
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], sig); err != nil {
		return nil, ErrSignatureInvalid
	}

	payB, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrTokenMalformed
	}
	var c Claims
	if err := json.Unmarshal(payB, &c); err != nil {
		return nil, ErrTokenMalformed
	}
	var raw struct {
		Aud json.RawMessage `json:"aud"`
	}
	_ = json.Unmarshal(payB, &raw)
	if len(raw.Aud) > 0 {
		var one string
		if json.Unmarshal(raw.Aud, &one) == nil {
			c.Aud = []string{one}
		} else if json.Unmarshal(raw.Aud, &c.Aud) != nil {
			return nil, ErrTokenMalformed
		}
	}
	// Token sem exp seria eterno; sem iss poderia vir de outro emissor que
	// compartilhe a chave. Ambos obrigatórios.
	if c.Exp == 0 || time.Now().Unix() >= c.Exp {
		return nil, ErrTokenExpired
	}
	if c.Iss != v.Issuer {
		return nil, ErrIssuerMismatch
	}
	if v.Audience != "" && !slices.Contains(c.Aud, v.Audience) {
		return nil, ErrAudienceMismatch
	}
	return &c, nil
}

// key devolve a chave do kid; kid desconhecido força um refresh do JWKS
// (com folga mínima de 30s entre refreshes para não martelar o IdP).
func (v *Verifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	k, ok := v.keys[kid]
	last := v.last
	v.mu.RUnlock()
	if ok {
		return k, nil
	}
	if time.Since(last) < 30*time.Second {
		return nil, ErrKeyNotFound
	}
	if err := v.refresh(ctx); err != nil {
		return nil, err
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	if k, ok := v.keys[kid]; ok {
		return k, nil
	}
	return nil, ErrKeyNotFound
}

func (v *Verifier) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.Issuer+"/.well-known/jwks.json", nil)
	if err != nil {
		return fmt.Errorf("vertikonid: request jwks: %w", err)
	}
	resp, err := v.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("vertikonid: fetch jwks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("vertikonid: jwks status %d", resp.StatusCode)
	}
	var doc struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("vertikonid: decode jwks: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, jk := range doc.Keys {
		if jk.Kty != "RSA" {
			continue
		}
		nb, err := base64.RawURLEncoding.DecodeString(jk.N)
		if err != nil {
			continue
		}
		eb, err := base64.RawURLEncoding.DecodeString(jk.E)
		if err != nil {
			continue
		}
		keys[jk.Kid] = &rsa.PublicKey{
			N: new(big.Int).SetBytes(nb),
			E: int(new(big.Int).SetBytes(eb).Int64()),
		}
	}
	v.mu.Lock()
	v.keys = keys
	v.last = time.Now()
	v.mu.Unlock()
	return nil
}
