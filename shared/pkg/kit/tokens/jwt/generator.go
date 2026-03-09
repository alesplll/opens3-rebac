package jwt

import (
	"context"
	"fmt"
	"time"

	"github.com/alesplll/opens3-rebac/shared/pkg/kit/logger"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/tokens"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type JWTGeneratorConfig interface {
	RefreshTokenSecretKey() string
	AccessTokenSecretKey() string

	RefreshTokenExpiration() time.Duration
	AccessTokenExpiration() time.Duration
}

type JWTGenerator struct {
	cfg JWTGeneratorConfig
}

func NewJWTGenerator(cfg JWTGeneratorConfig) tokens.TokenGenerator {
	return &JWTGenerator{
		cfg: cfg,
	}
}

func (j *JWTGenerator) GenerateAccessToken(ctx context.Context, info tokens.UserInfo) (string, error) {
	secretKey := []byte(j.cfg.AccessTokenSecretKey())
	duration := j.cfg.AccessTokenExpiration()
	return j.generateToken(ctx, info, duration, AccessToken, secretKey)
}

func (j *JWTGenerator) GenerateRefreshToken(ctx context.Context, info tokens.UserInfo) (string, error) {
	secretKey := []byte(j.cfg.RefreshTokenSecretKey())
	duration := j.cfg.RefreshTokenExpiration()
	return j.generateToken(ctx, info, duration, RefreshToken, secretKey)
}

func (j *JWTGenerator) generateToken(ctx context.Context, info tokens.UserInfo, duration time.Duration, tokenType TokenType, secretKey []byte) (string, error) {
	if len(secretKey) == 0 {
		return "", fmt.Errorf("%s secret key is empty", tokenType)
	}

	now := time.Now()
	claims := tokens.UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
		UserId:    info.GetUserID(),
		Email:     info.GetEmail(),
		TokenType: string(tokenType),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(secretKey)
	if err != nil {
		logger.Error(ctx, "failed to sign token", zap.String("tokenType", string(tokenType)), zap.Error(err))
		return "", fmt.Errorf("failed to sign %s token: %w", tokenType, err)
	}

	return signedToken, nil
}
