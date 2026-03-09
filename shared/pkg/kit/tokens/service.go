package tokens

import (
	"context"
)

type TokenGenerator interface {
	GenerateAccessToken(context.Context, UserInfo) (string, error)
	GenerateRefreshToken(context.Context, UserInfo) (string, error)
}

type TokenVerifier interface {
	VerifyAccessToken(context.Context, string) (*UserClaims, error)
	VerifyRefreshToken(context.Context, string) (*UserClaims, error)
}

type TokenService interface {
	TokenGenerator
	TokenVerifier
}
