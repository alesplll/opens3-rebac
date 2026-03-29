package auth

import (
	"context"

	desc_auth "github.com/alesplll/opens3-rebac/shared/pkg/go/auth/v1"
)

func (h *authHandler) GetRefreshToken(ctx context.Context, req *desc_auth.GetRefreshTokenRequest) (*desc_auth.GetRefreshTokenResponse, error) {
	new_refresh_token, err := h.service.GetRefreshToken(ctx, req.GetRefreshToken())
	return &desc_auth.GetRefreshTokenResponse{
		RefreshToken: new_refresh_token,
	}, err
}
