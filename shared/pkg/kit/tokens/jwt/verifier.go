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

type JWTConfigVerifier interface {
	RefreshTokenSecretKey() string
	AccessTokenSecretKey() string
}

type JWTVerifier struct {
	cfg JWTConfigVerifier
}

func NewJWTVerifier(cfg JWTConfigVerifier) tokens.TokenVerifier {
	return &JWTVerifier{
		cfg: cfg,
	}
}

func (j *JWTVerifier) VerifyAccessToken(ctx context.Context, tokenStr string) (*tokens.UserClaims, error) {
	secretKey := []byte(j.cfg.AccessTokenSecretKey())
	claims, err := j.verifyToken(ctx, tokenStr, secretKey, AccessToken)
	if err != nil {
		return nil, fmt.Errorf("access token verification failed: %w", err)
	}
	return claims, nil
}

func (j *JWTVerifier) VerifyRefreshToken(ctx context.Context, tokenStr string) (*tokens.UserClaims, error) {
	secretKey := []byte(j.cfg.RefreshTokenSecretKey())
	claims, err := j.verifyToken(ctx, tokenStr, secretKey, RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("refresh token verification failed: %w", err)
	}
	return claims, nil
}

func (j *JWTVerifier) verifyToken(ctx context.Context, tokenStr string, secretKey []byte, expectedType TokenType) (*tokens.UserClaims, error) {
	if tokenStr == "" {
		return nil, fmt.Errorf("token string is empty")
	}

	token, err := jwt.ParseWithClaims(
		tokenStr,
		&tokens.UserClaims{},
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				err := fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				logger.Info(ctx, "unexpected signing method", zap.Error(err))
				return nil, err
			}
			return secretKey, nil
		},
	)
	if err != nil {
		switch {
		case err == jwt.ErrTokenExpired:
			return nil, fmt.Errorf("token has expired")
		case err == jwt.ErrTokenNotValidYet:
			return nil, fmt.Errorf("token is not valid yet")
		case err == jwt.ErrTokenMalformed:
			return nil, fmt.Errorf("token is malformed")
		default:
			return nil, fmt.Errorf("invalid token: %w", err)
		}
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	claims, ok := token.Claims.(*tokens.UserClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims type")
	}

	if claims.TokenType != string(expectedType) {
		return nil, fmt.Errorf("invalid token type: expected %s, got %s", expectedType, claims.TokenType)
	}

	if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("token has expired")
	}

	return claims, nil
}
