package validation

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type noopLogger struct{}

func (noopLogger) Info(context.Context, string, ...zap.Field)  {}
func (noopLogger) Error(context.Context, string, ...zap.Field) {}

func TestHandleError_UnexpectedEOFBecomesInvalidArgument(t *testing.T) {
	t.Parallel()

	err := handleError(context.Background(), io.ErrUnexpectedEOF, noopLogger{})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Contains(t, err.Error(), io.ErrUnexpectedEOF.Error())
}
