package auth

import (
	"context"

	desc_auth "github.com/alesplll/opens3-rebac/shared/pkg/auth/v1"
)

func (h *authHandler) GetAccessToken(ctx context.Context, req *desc_auth.GetAccessTokenRequest) (*desc_auth.GetAccessTokenResponse, error) {
	new_access_token, err := h.service.GetAccessToken(ctx, req.GetRefreshToken())
	return &desc_auth.GetAccessTokenResponse{
		AccessToken: new_access_token,
	}, err
}
