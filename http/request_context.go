package http

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/vertikon/sdk-hulk.vertikon.com.br/state"
)

type ctxKey string

const (
	ctxKeyTenantID ctxKey = "hulk_tenant_id"
	ctxKeyGroupID  ctxKey = "hulk_group_id"
	ctxKeyStoreID  ctxKey = "hulk_store_id"
	ctxKeyUserID   ctxKey = "hulk_user_id"
)

var (
	ErrTenantIDMissing = errors.New("tenant id missing")
	ErrTenantIDInvalid = errors.New("tenant id invalid")
	ErrUserIDMissing   = errors.New("user id missing")
	ErrUserIDInvalid   = errors.New("user id invalid")
)

func TenantIDFromContext(ctx context.Context) (uuid.UUID, error) {
	if ctx == nil {
		return uuid.Nil, ErrTenantIDMissing
	}
	if v := ctx.Value(ctxKeyTenantID); v != nil {
		if id, ok := v.(uuid.UUID); ok && id != uuid.Nil {
			return id, nil
		}
	}
	return uuid.Nil, ErrTenantIDMissing
}

func GroupIDFromContext(ctx context.Context) (uuid.UUID, error) {
	if ctx == nil {
		return uuid.Nil, nil
	}
	if v := ctx.Value(ctxKeyGroupID); v != nil {
		if id, ok := v.(uuid.UUID); ok {
			return id, nil
		}
	}
	return uuid.Nil, nil
}

func StoreIDFromContext(ctx context.Context) (uuid.UUID, error) {
	if ctx == nil {
		return uuid.Nil, nil
	}
	if v := ctx.Value(ctxKeyStoreID); v != nil {
		if id, ok := v.(uuid.UUID); ok {
			return id, nil
		}
	}
	return uuid.Nil, nil
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	if ctx == nil {
		return uuid.Nil, nil
	}
	if v := ctx.Value(ctxKeyUserID); v != nil {
		if id, ok := v.(uuid.UUID); ok {
			return id, nil
		}
	}
	return uuid.Nil, nil
}

func RequireUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	id, _ := UserIDFromContext(ctx)
	if id == uuid.Nil {
		return uuid.Nil, ErrUserIDMissing
	}
	return id, nil
}

func WithTenantID(ctx context.Context, tenantID uuid.UUID) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxKeyTenantID, tenantID)
}

// WithTenantSlug injeta o tenant vindo de um PATH público (ex.: LTI /t/:tenant/...)
// no contexto RLS. Só uuids válidos contam — slugs como "global" não viram
// contexto (mesma regra do hook TenantFromContext).
func WithTenantSlug(ctx context.Context, tenant string) context.Context {
	if tid, err := uuid.Parse(tenant); err == nil && tid != uuid.Nil {
		return WithTenantID(ctx, tid)
	}
	return ctx
}

func WithGroupID(ctx context.Context, groupID uuid.UUID) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxKeyGroupID, groupID)
}

func WithStoreID(ctx context.Context, storeID uuid.UUID) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxKeyStoreID, storeID)
}

func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxKeyUserID, userID)
}

// init injeta o leitor de tenant no pacote state (encanamento RLS F1.2):
// o state não pode importar http (peso/eco de deps), então recebe o hook.
//
// Fallback pela chave STRING "tenant_id": os consumers NATS (engine do wa-core,
// signal-classifier, etc.) montam o contexto com context.WithValue(ctx,
// "tenant_id", s) — fora do middleware HTTP não existe a chave tipada, e sem o
// fallback essas queries rodavam SEM set_config → RLS devolvia 0 linhas e os
// handlers saíam calados (incidente 2026-07-16: ai-bot mudo desde o flip 14:02).
// Só uuids válidos contam — "default"/"global" não viram contexto RLS.
func init() {
	state.TenantFromContext = func(ctx context.Context) (string, bool) {
		if tid, err := TenantIDFromContext(ctx); err == nil {
			return tid.String(), true
		}
		if s, ok := ctx.Value("tenant_id").(string); ok {
			if tid, err := uuid.Parse(s); err == nil && tid != uuid.Nil {
				return tid.String(), true
			}
		}
		return "", false
	}
}
