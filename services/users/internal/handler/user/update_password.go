package user

import (
	"context"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/user/v1"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/contextx/ipctx"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (h *handler) UpdatePassword(ctx context.Context, req *desc.UpdatePasswordRequest) (*emptypb.Empty, error) {
	ctx = ipctx.InjectIp(ctx)
	return &emptypb.Empty{}, h.service.UpdatePassword(ctx, req.GetUserId(), req.GetPassword(), req.GetPasswordConfirm())
}
