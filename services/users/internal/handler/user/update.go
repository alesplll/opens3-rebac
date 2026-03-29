package user

import (
	"context"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/user/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (h *handler) Update(ctx context.Context, req *desc.UpdateRequest) (*emptypb.Empty, error) {
	var name *string
	if req.GetName() != nil {
		name = &req.GetName().Value
	}

	var email *string
	if req.GetEmail() != nil {
		email = &req.GetEmail().Value
	}

	return &emptypb.Empty{}, h.service.Update(ctx, req.GetUserId(), name, email)
}
