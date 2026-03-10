package validation

import (
	customCodes "github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys/codes"
	grpcCodes "google.golang.org/grpc/codes"
)

func toGRPCCode(code customCodes.Code) grpcCodes.Code {
	var res grpcCodes.Code

	switch code {
	case customCodes.OK:
		res = grpcCodes.OK
	case customCodes.Canceled:
		res = grpcCodes.Canceled
	case customCodes.InvalidArgument:
		res = grpcCodes.InvalidArgument
	case customCodes.DeadlineExceeded:
		res = grpcCodes.DeadlineExceeded
	case customCodes.NotFound:
		res = grpcCodes.NotFound
	case customCodes.AlreadyExists:
		res = grpcCodes.AlreadyExists
	case customCodes.PermissionDenied:
		res = grpcCodes.PermissionDenied
	case customCodes.ResourceExhausted:
		res = grpcCodes.ResourceExhausted
	case customCodes.FailedPrecondition:
		res = grpcCodes.FailedPrecondition
	case customCodes.Aborted:
		res = grpcCodes.Aborted
	case customCodes.OutOfRange:
		res = grpcCodes.OutOfRange
	case customCodes.Unimplemented:
		res = grpcCodes.Unimplemented
	case customCodes.Internal:
		res = grpcCodes.Internal
	case customCodes.Unavailable:
		res = grpcCodes.Unavailable
	case customCodes.DataLoss:
		res = grpcCodes.DataLoss
	case customCodes.Unauthenticated:
		res = grpcCodes.Unauthenticated
	default:
		res = grpcCodes.Unknown
	}

	return res
}
