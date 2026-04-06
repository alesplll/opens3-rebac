package component

import (
	"context"
	"testing"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck_Serving(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resp, err := client.HealthCheck(ctx, &desc.HealthCheckRequest{Service: ""})
	require.NoError(t, err)
	require.Equal(t, desc.HealthCheckResponse_SERVING, resp.GetStatus())
}
