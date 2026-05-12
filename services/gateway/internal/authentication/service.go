package authentication

import (
	"context"
	"strings"

	domainerrors "github.com/alesplll/opens3-rebac/services/gateway/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/tokens"
)

type AccessTokenVerifier interface {
	VerifyAccessToken(context.Context, string) (*tokens.UserClaims, error)
}

type Service interface {
	ClaimsFromAccessToken(ctx context.Context, token string) (*tokens.UserClaims, error)
}

type authService struct {
	verifier AccessTokenVerifier
}

func NewService(verifier AccessTokenVerifier) Service {
	return &authService{verifier: verifier}
}

func (s *authService) ClaimsFromAccessToken(ctx context.Context, token string) (*tokens.UserClaims, error) {
	if strings.TrimSpace(token) == "" {
		return nil, domainerrors.ErrUnauthorized
	}

	claims, err := s.verifier.VerifyAccessToken(ctx, token)
	if err != nil || claims == nil || strings.TrimSpace(claims.UserId) == "" {
		return nil, domainerrors.ErrUnauthorized
	}

	return claims, nil
}
