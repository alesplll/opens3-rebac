package jwt

import (
	"context"
	"time"

	"github.com/alesplll/opens3-rebac/shared/pkg/kit/tokens"
)

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

type JWTServiceConfig interface {
	RefreshTokenSecretKey() string
	AccessTokenSecretKey() string

	RefreshTokenExpiration() time.Duration
	AccessTokenExpiration() time.Duration
}

type JWTService struct {
	cfg       JWTServiceConfig
	generator tokens.TokenGenerator
	verifier  tokens.TokenVerifier
}

func NewJWTService(cfg JWTServiceConfig) tokens.TokenService {
	return &JWTService{
		cfg:       cfg,
		generator: NewJWTGenerator(cfg),
		verifier:  NewJWTVerifier(cfg),
	}
}

func (s *JWTService) GenerateAccessToken(ctx context.Context, info tokens.UserInfo) (string, error) {
	return s.generator.GenerateAccessToken(ctx, info)
}

func (s *JWTService) GenerateRefreshToken(ctx context.Context, info tokens.UserInfo) (string, error) {
	return s.generator.GenerateRefreshToken(ctx, info)
}

func (s *JWTService) VerifyAccessToken(ctx context.Context, tokenStr string) (*tokens.UserClaims, error) {
	return s.verifier.VerifyAccessToken(ctx, tokenStr)
}

func (s *JWTService) VerifyRefreshToken(ctx context.Context, tokenStr string) (*tokens.UserClaims, error) {
	return s.verifier.VerifyRefreshToken(ctx, tokenStr)
}
