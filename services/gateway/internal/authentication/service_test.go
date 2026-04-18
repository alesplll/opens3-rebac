package authentication

import (
	"context"
	"errors"
	"testing"

	domainerrors "github.com/alesplll/opens3-rebac/services/gateway/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/tokens"
	"github.com/stretchr/testify/require"
)

type stubVerifier struct {
	verifyAccessTokenFn func(ctx context.Context, token string) (*tokens.UserClaims, error)
}

func (s *stubVerifier) VerifyAccessToken(ctx context.Context, token string) (*tokens.UserClaims, error) {
	if s.verifyAccessTokenFn == nil {
		panic("unexpected call")
	}
	return s.verifyAccessTokenFn(ctx, token)
}

func TestClaimsFromAccessToken(t *testing.T) {
	t.Parallel()

	svc := NewService(&stubVerifier{
		verifyAccessTokenFn: func(ctx context.Context, token string) (*tokens.UserClaims, error) {
			require.Equal(t, "access-token", token)
			return &tokens.UserClaims{UserId: "user-1"}, nil
		},
	})

	claims, err := svc.ClaimsFromAccessToken(context.Background(), "access-token")

	require.NoError(t, err)
	require.Equal(t, "user-1", claims.UserId)
}

func TestClaimsFromAccessTokenRejectsInvalidClaims(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		token  string
		claims *tokens.UserClaims
		err    error
	}{
		{name: "empty token", token: "   "},
		{name: "verifier error", token: "access-token", err: errors.New("boom")},
		{name: "nil claims", token: "access-token"},
		{name: "empty user id", token: "access-token", claims: &tokens.UserClaims{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := NewService(&stubVerifier{
				verifyAccessTokenFn: func(ctx context.Context, token string) (*tokens.UserClaims, error) {
					return tt.claims, tt.err
				},
			})

			claims, err := svc.ClaimsFromAccessToken(context.Background(), tt.token)

			require.Nil(t, claims)
			require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
		})
	}
}
