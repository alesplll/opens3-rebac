package users

import (
	"context"

	grpcclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc"
	userv1 "github.com/alesplll/opens3-rebac/shared/pkg/go/user/v1"
)

type client struct {
	client userv1.UserV1Client
}

func NewClient(grpcClient userv1.UserV1Client) grpcclient.UsersClient {
	return &client{client: grpcClient}
}

func (c *client) ValidateCredentials(ctx context.Context, req *userv1.ValidateCredentialsRequest) (*userv1.ValidateCredentialsResponse, error) {
	return c.client.ValidateCredentials(ctx, req)
}
