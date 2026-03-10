package user

import (
	"context"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/user/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (h *handler) Delete(ctx context.Context, req *desc.DeleteRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, h.service.Delete(ctx, req.GetUserId())
}
