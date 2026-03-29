package user

import (
	"context"

	conventer "github.com/alesplll/opens3-rebac/services/users/internal/conventer/user"
	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/user/v1"
)

func (h *handler) Create(ctx context.Context, req *desc.CreateRequest) (*desc.CreateResponse, error) {
	userID, err := h.service.Create(ctx, conventer.FromProtoToModelUserInfo(req.GetUserInfo()), req.GetPassword(), req.GetPasswordConfirm())
	if err != nil {
		return nil, err
	}

	return &desc.CreateResponse{
		Id: userID,
	}, nil
}
