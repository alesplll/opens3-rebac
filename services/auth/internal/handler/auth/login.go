package auth

import (
	"context"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/auth/v1"
)

func (h *authHandler) Login(ctx context.Context, req *desc.LoginRequest) (*desc.LoginResponse, error) {
	refresh_token, err := h.service.Login(ctx, req.GetEmail(), req.GetPassword())
	return &desc.LoginResponse{
		RefreshToken: refresh_token,
	}, err
}
