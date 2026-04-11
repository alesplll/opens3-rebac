package metadata

import (
	"context"
	"testing"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerReturnsNotImplementedForAllMethods(t *testing.T) {
	type testCase struct {
		name string
		call func(t *testing.T, h metadatav1.MetadataServiceServer)
	}

	tests := []testCase{
		{
			name: "CreateBucket",
			call: func(t *testing.T, h metadatav1.MetadataServiceServer) {
				res, err := h.CreateBucket(context.Background(), &metadatav1.CreateBucketRequest{})
				assertNotImplemented(t, res, err, "method CreateBucket not implemented")
			},
		},
		{
			name: "DeleteBucket",
			call: func(t *testing.T, h metadatav1.MetadataServiceServer) {
				res, err := h.DeleteBucket(context.Background(), &metadatav1.DeleteBucketRequest{})
				assertNotImplemented(t, res, err, "method DeleteBucket not implemented")
			},
		},
		{
			name: "GetBucket",
			call: func(t *testing.T, h metadatav1.MetadataServiceServer) {
				res, err := h.GetBucket(context.Background(), &metadatav1.GetBucketRequest{})
				assertNotImplemented(t, res, err, "method GetBucket not implemented")
			},
		},
		{
			name: "ListBuckets",
			call: func(t *testing.T, h metadatav1.MetadataServiceServer) {
				res, err := h.ListBuckets(context.Background(), &metadatav1.ListBucketsRequest{})
				assertNotImplemented(t, res, err, "method ListBuckets not implemented")
			},
		},
		{
			name: "HeadBucket",
			call: func(t *testing.T, h metadatav1.MetadataServiceServer) {
				res, err := h.HeadBucket(context.Background(), &metadatav1.HeadBucketRequest{})
				assertNotImplemented(t, res, err, "method HeadBucket not implemented")
			},
		},
		{
			name: "CreateObjectVersion",
			call: func(t *testing.T, h metadatav1.MetadataServiceServer) {
				res, err := h.CreateObjectVersion(context.Background(), &metadatav1.CreateObjectVersionRequest{})
				assertNotImplemented(t, res, err, "method CreateObjectVersion not implemented")
			},
		},
		{
			name: "GetObjectMeta",
			call: func(t *testing.T, h metadatav1.MetadataServiceServer) {
				res, err := h.GetObjectMeta(context.Background(), &metadatav1.GetObjectMetaRequest{})
				assertNotImplemented(t, res, err, "method GetObjectMeta not implemented")
			},
		},
		{
			name: "DeleteObjectMeta",
			call: func(t *testing.T, h metadatav1.MetadataServiceServer) {
				res, err := h.DeleteObjectMeta(context.Background(), &metadatav1.DeleteObjectMetaRequest{})
				assertNotImplemented(t, res, err, "method DeleteObjectMeta not implemented")
			},
		},
		{
			name: "ListObjects",
			call: func(t *testing.T, h metadatav1.MetadataServiceServer) {
				res, err := h.ListObjects(context.Background(), &metadatav1.ListObjectsRequest{})
				assertNotImplemented(t, res, err, "method ListObjects not implemented")
			},
		},
		{
			name: "HealthCheck",
			call: func(t *testing.T, h metadatav1.MetadataServiceServer) {
				res, err := h.HealthCheck(context.Background(), &metadatav1.HealthCheckRequest{})
				assertNotImplemented(t, res, err, "method HealthCheck not implemented")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.call(t, NewHandler())
		})
	}
}

func assertNotImplemented[T any](t *testing.T, res *T, err error, message string) {
	t.Helper()

	require.Nil(t, res)
	require.Error(t, err)

	grpcStatus, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Unimplemented, grpcStatus.Code())
	require.Equal(t, message, grpcStatus.Message())
}
