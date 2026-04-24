package domain_errors

import (
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys/codes"
)

var (
	ErrBucketNotFound      = sys.NewCommonError("bucket not found", codes.NotFound)
	ErrObjectNotFound      = sys.NewCommonError("object not found", codes.NotFound)
	ErrBucketAlreadyExists = sys.NewCommonError("bucket already exists", codes.AlreadyExists)
	ErrBucketNotEmpty      = sys.NewCommonError("bucket is not empty", codes.FailedPrecondition)
	ErrInternal            = sys.NewCommonError("internal error", codes.Internal)
)
