package user

import (
	"context"

	conventer "github.com/alesplll/opens3-rebac/services/users/internal/conventer/user"
	desc "github.com/alesplll/opens3-rebac/shared/pkg/user/v1"
)

func (h *handler) Get(ctx context.Context, req *desc.GetRequest) (*desc.GetResponse, error) {
	user, err := h.service.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &desc.GetResponse{
		User: conventer.FromModelToProtoUser(*user),
	}, nil
}
