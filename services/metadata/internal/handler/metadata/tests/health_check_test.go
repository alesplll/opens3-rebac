package tests

import (
	"context"
	"testing"

	"github.com/alesplll/opens3-rebac/services/metadata/pkg/mocks"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck(t *testing.T) {
	t.Run("serving", func(t *testing.T) {
		ctx := context.Background()
		mc := minimock.NewController(t)

		objectServiceMock := mocks.NewObjectServiceMock(mc)
		objectServiceMock.HealthCheckMock.Expect(ctx).Return(true, true)

		handler := newHandler(nil, objectServiceMock)

		res, err := handler.HealthCheck(ctx, &metadatav1.HealthCheckRequest{})

		require.NoError(t, err)
		require.Equal(t, &metadatav1.HealthCheckResponse{
			Status: metadatav1.HealthCheckResponse_SERVING,
		}, res)
	})

	t.Run("not serving when postgres is down", func(t *testing.T) {
		mc := minimock.NewController(t)
		objectServiceMock := mocks.NewObjectServiceMock(mc)
		objectServiceMock.HealthCheckMock.Expect(context.Background()).Return(false, true)

		handler := newHandler(nil, objectServiceMock)

		res, err := handler.HealthCheck(context.Background(), &metadatav1.HealthCheckRequest{})

		require.NoError(t, err)
		require.Equal(t, &metadatav1.HealthCheckResponse{
			Status: metadatav1.HealthCheckResponse_NOT_SERVING,
		}, res)
	})

	t.Run("not serving when kafka is down", func(t *testing.T) {
		mc := minimock.NewController(t)
		objectServiceMock := mocks.NewObjectServiceMock(mc)
		objectServiceMock.HealthCheckMock.Expect(context.Background()).Return(true, false)

		handler := newHandler(nil, objectServiceMock)

		res, err := handler.HealthCheck(context.Background(), &metadatav1.HealthCheckRequest{})

		require.NoError(t, err)
		require.Equal(t, &metadatav1.HealthCheckResponse{
			Status: metadatav1.HealthCheckResponse_NOT_SERVING,
		}, res)
	})
}
