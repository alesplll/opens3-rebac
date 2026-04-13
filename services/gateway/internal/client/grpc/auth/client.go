package auth

import (
	"context"

	grpcclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc"
	authv1 "github.com/alesplll/opens3-rebac/shared/pkg/go/auth/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

type client struct {
	client authv1.AuthV1Client
}

func NewClient(grpcClient authv1.AuthV1Client) grpcclient.AuthClient {
	return &client{client: grpcClient}
}

func (c *client) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	return c.client.Login(ctx, req)
}

func (c *client) GetRefreshToken(ctx context.Context, req *authv1.GetRefreshTokenRequest) (*authv1.GetRefreshTokenResponse, error) {
	return c.client.GetRefreshToken(ctx, req)
}

func (c *client) GetAccessToken(ctx context.Context, req *authv1.GetAccessTokenRequest) (*authv1.GetAccessTokenResponse, error) {
	return c.client.GetAccessToken(ctx, req)
}

func (c *client) ValidateToken(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	return c.client.ValidateToken(ctx, req)
}
