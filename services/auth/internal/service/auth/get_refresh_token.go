package auth

import (
	"context"

	"github.com/alesplll/opens3-rebac/services/auth/internal/model"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/contextx/claimsctx"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys/codes"
	"go.uber.org/zap"
)

func (s *authService) GetRefreshToken(ctx context.Context, old_refresh_token string) (string, error) {
	claims, err := s.tokenService.VerifyRefreshToken(ctx, old_refresh_token)
	if err != nil {
		return "", sys.NewCommonError("ivalid refresh token", codes.Unauthenticated)
	}

	ctx = claimsctx.InjectUserEmail(ctx, claims.Email)
	ctx = claimsctx.InjectUserID(ctx, claims.UserId)

	new_refresh_token, err := s.tokenService.GenerateRefreshToken(ctx, model.UserInfo{
		UserId: claims.UserId,
		Email:  claims.Email,
	})
	if err != nil {
		logger.Error(ctx, "failed to generate refresh token", zap.Error(err))
		return "", err
	}

	return new_refresh_token, nil
}
