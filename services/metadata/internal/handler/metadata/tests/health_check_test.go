package tests

import (
	"context"
	"testing"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck(t *testing.T) {
	t.Run("serving", func(t *testing.T) {
		ctx := context.Background()

		handler := newHandler(nil, &objectServiceStub{
			healthCheckFunc: func(gotCtx context.Context) (bool, bool) {
				require.Equal(t, ctx, gotCtx)
				return true, true
			},
		})

		res, err := handler.HealthCheck(ctx, &metadatav1.HealthCheckRequest{})

		require.NoError(t, err)
		require.Equal(t, &metadatav1.HealthCheckResponse{
			Status: metadatav1.HealthCheckResponse_SERVING,
		}, res)
	})

	t.Run("not serving when postgres is down", func(t *testing.T) {
		handler := newHandler(nil, &objectServiceStub{
			healthCheckFunc: func(context.Context) (bool, bool) {
				return false, true
			},
		})

		res, err := handler.HealthCheck(context.Background(), &metadatav1.HealthCheckRequest{})

		require.NoError(t, err)
		require.Equal(t, &metadatav1.HealthCheckResponse{
			Status: metadatav1.HealthCheckResponse_NOT_SERVING,
		}, res)
	})

	t.Run("not serving when kafka is down", func(t *testing.T) {
		handler := newHandler(nil, &objectServiceStub{
			healthCheckFunc: func(context.Context) (bool, bool) {
				return true, false
			},
		})

		res, err := handler.HealthCheck(context.Background(), &metadatav1.HealthCheckRequest{})

		require.NoError(t, err)
		require.Equal(t, &metadatav1.HealthCheckResponse{
			Status: metadatav1.HealthCheckResponse_NOT_SERVING,
		}, res)
	})
}
