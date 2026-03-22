package sys

import (
	"github.com/pkg/errors"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys/codes"
)

type CommonError struct {
	msg  string
	code codes.Code
}

func NewCommonError(msg string, code codes.Code) *CommonError {
	return &CommonError{msg, code}
}

func (r *CommonError) Error() string {
	return r.msg
}

func (r *CommonError) Code() codes.Code {
	return r.code
}

func IsCommonError(err error) bool {
	var ce *CommonError
	return errors.As(err, &ce)
}

func GetCommonError(err error) *CommonError {
	var ce *CommonError
	if !errors.As(err, &ce) {
		return nil
	}

	return ce
}
