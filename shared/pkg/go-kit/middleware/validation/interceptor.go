package validation

import (
	"context"
	"io"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys/validate"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	grpcCodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCStatus interface {
	GRPCStatus() *status.Status
}

type Logger interface {
	Info(ctx context.Context, msg string, fields ...zap.Field)
	Error(ctx context.Context, msg string, fields ...zap.Field)
}

// handleError maps commonErr, validationErr, and context errors to gRPC status errors and logs at appropriate levels
func handleError(ctx context.Context, err error, logger Logger) error {
	if nil == err {
		return nil
	}
	switch {
	case sys.IsCommonError(err):
		commonErr := sys.GetCommonError(err)
		code := toGRPCCode(commonErr.Code())

		logger.Info(ctx, "error interceptor handle common error", zap.Error(err))
		return status.Error(code, commonErr.Error())

	case validate.IsValidationError(err):
		logger.Info(ctx, "error interceptor handle validation error", zap.Error(err))
		return status.Error(grpcCodes.InvalidArgument, err.Error())
	default:
		var se GRPCStatus
		if errors.As(err, &se) {
			return se.GRPCStatus().Err()
		} else {
			if errors.Is(err, io.ErrUnexpectedEOF) {
				logger.Info(ctx, "error interceptor handle unexpected eof", zap.Error(err))
				return status.Error(grpcCodes.InvalidArgument, err.Error())
			} else if errors.Is(err, context.DeadlineExceeded) {
				logger.Info(ctx, "error interceptor deadlineExceeded error", zap.Error(err))
				return status.Error(grpcCodes.DeadlineExceeded, err.Error())
			} else if errors.Is(err, context.Canceled) {
				logger.Info(ctx, "error interceptor handle canceled context", zap.Error(err))
				return status.Error(grpcCodes.Canceled, err.Error())
			} else {
				logger.Error(ctx, "error interceptor internal error", zap.Error(err))
				return status.Error(grpcCodes.Internal, "internal error")
			}
		}
	}
}

func ErrorCodesUnaryInterceptor(logger Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (res any, err error) {
		res, err = handler(ctx, req)
		return res, handleError(ctx, err, logger)
	}
}

func ErrorCodesStreamInterceptor(logger Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, ss)
		return handleError(ss.Context(), err, logger)
	}
}
