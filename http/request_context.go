package http

import (
	"context"
	"errors"

	"github.com/google/uuid"
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
