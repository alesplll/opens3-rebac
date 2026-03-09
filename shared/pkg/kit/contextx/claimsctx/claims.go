package claimsctx

import (
	"context"

	"github.com/alesplll/opens3-rebac/shared/pkg/kit/contextx"
)

const (
	UserEmailKey contextx.CtxKey = "user_email"
	UserIDKey    contextx.CtxKey = "user_id"
)

func InjectUserEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, UserEmailKey, email)
}

func InjectUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

func ExtractUserEmail(ctx context.Context) (string, bool) {
	if email, ok := ctx.Value(UserEmailKey).(string); ok {
		return email, true
	}
	return "unknown", false
}

func ExtractUserID(ctx context.Context) (string, bool) {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID, true
	}
	return "", false
}
