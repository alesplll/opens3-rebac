package domainerrors

import (
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys/codes"
)

var (
	ErrBlobNotFound   = sys.NewCommonError("blob not found", codes.NotFound)
	ErrUploadNotFound = sys.NewCommonError("upload not found", codes.NotFound)

	ErrInvalidPartNumber = sys.NewCommonError("invalid part number", codes.InvalidArgument)
	ErrChecksumMismatch  = sys.NewCommonError("checksum mismatch", codes.InvalidArgument)

	ErrDiskFull = sys.NewCommonError("insufficient disk space", codes.ResourceExhausted)

	ErrInternal = sys.NewCommonError("internal error", codes.Internal)
)
