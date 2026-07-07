package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func parseUUIDHeader(raw string) (uuid.UUID, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// RequestContextMiddleware injects tenancy/user identifiers into request context.
// Sources (priority):
// - Headers: X-Tenant-Id, X-Group-Id, X-Store-Id, X-User-Id
// - Auth user (if NewAuthMiddleware ran): security.User in echo context
//
// Notes:
// - Does not require tenant; downstream services may enforce it.
// - Invalid UUIDs return 400 (fail-fast).
func RequestContextMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			base := req.Context()

			tenantID, tenantOK := parseUUIDHeader(req.Header.Get("X-Tenant-Id"))
			groupID, groupOK := parseUUIDHeader(req.Header.Get("X-Group-Id"))
			storeID, storeOK := parseUUIDHeader(req.Header.Get("X-Store-Id"))
			userID, userOK := parseUUIDHeader(req.Header.Get("X-User-Id"))

			// Allow compatibility: if tenant header missing, accept X-Group-Id as tenant.
			if !tenantOK && groupOK {
				tenantID, tenantOK = groupID, true
			}

			// If auth middleware provided a user, use it as fallback.
			if user := GetUserFromContext(c); user != nil {
				if !tenantOK && user.TenantID != uuid.Nil {
					tenantID, tenantOK = user.TenantID, true
				}
				if !userOK {
					if parsed, err := uuid.Parse(strings.TrimSpace(user.ID)); err == nil {
						userID, userOK = parsed, true
					}
				}
			}

			// Invalid UUID strings: reject early.
			for _, kv := range []struct {
				raw string
				ok  bool
				hdr string
			}{
				{raw: req.Header.Get("X-Tenant-Id"), ok: tenantOK || strings.TrimSpace(req.Header.Get("X-Tenant-Id")) == "", hdr: "X-Tenant-Id"},
				{raw: req.Header.Get("X-Group-Id"), ok: groupOK || strings.TrimSpace(req.Header.Get("X-Group-Id")) == "", hdr: "X-Group-Id"},
				{raw: req.Header.Get("X-Store-Id"), ok: storeOK || strings.TrimSpace(req.Header.Get("X-Store-Id")) == "", hdr: "X-Store-Id"},
				{raw: req.Header.Get("X-User-Id"), ok: userOK || strings.TrimSpace(req.Header.Get("X-User-Id")) == "", hdr: "X-User-Id"},
			} {
				if strings.TrimSpace(kv.raw) != "" && !kv.ok {
					return c.JSON(http.StatusBadRequest, map[string]string{"error": kv.hdr + " must be a UUID"})
				}
			}

			ctx := base
			if tenantOK {
				ctx = context.WithValue(ctx, ctxKeyTenantID, tenantID)
			}
			if groupOK {
				ctx = context.WithValue(ctx, ctxKeyGroupID, groupID)
			}
			if storeOK {
				ctx = context.WithValue(ctx, ctxKeyStoreID, storeID)
			}
			if userOK {
				ctx = context.WithValue(ctx, ctxKeyUserID, userID)
			}

			// Attach updated context to request.
			if ctx != base {
				c.SetRequest(req.WithContext(ctx))
			}

			return next(c)
		}
	}
}

