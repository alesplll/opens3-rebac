package auth

import (
	"context"

	domainerrors "github.com/alesplll/opens3-rebac/services/auth/internal/errors/domain"
	"github.com/alesplll/opens3-rebac/services/auth/internal/model"
)

func (s *authService) GetAccessToken(ctx context.Context, refresh_token string) (string, error) {
	claims, err := s.tokenService.VerifyRefreshToken(ctx, refresh_token)
	if err != nil {
		return "", domainerrors.ErrInvalidRefreshToken
	}

	new_access_token, err := s.tokenService.GenerateAccessToken(ctx, model.UserInfo{
		UserId: claims.UserId,
		Email:  claims.Email,
	})
	if err != nil {
		return "", err
	}

	return new_access_token, nil
}
