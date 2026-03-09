package auth

import (
	"context"
	"strings"

	"github.com/alesplll/opens3-rebac/shared/pkg/kit/sys"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/sys/codes"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (h *authHandler) ValidateToken(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, sys.NewCommonError("metadata not provided", codes.Unauthenticated)
	}

	authHeader, ok := md["authorization"]
	if !ok || len(authHeader) == 0 {
		return nil, sys.NewCommonError("authorization header not provided", codes.Unauthenticated)
	}

	token := strings.TrimPrefix(authHeader[0], "Bearer ")

	err := h.service.ValidateToken(ctx, token)
	if err != nil {
		return &emptypb.Empty{}, sys.NewCommonError("invalid token", codes.Unauthenticated)
	}
	return &emptypb.Empty{}, nil
}
