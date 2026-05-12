package service

import "context"

type AuthService interface {
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
	RefreshAccessToken(ctx context.Context, req RefreshAccessTokenRequest) (*RefreshAccessTokenResponse, error)
	RefreshRefreshToken(ctx context.Context, req RefreshRefreshTokenRequest) (*RefreshRefreshTokenResponse, error)
}

type GatewayService interface {
	CreateBucket(ctx context.Context, req CreateBucketRequest) (*CreateBucketResponse, error)
	DeleteBucket(ctx context.Context, req DeleteBucketRequest) error
	ListBuckets(ctx context.Context, req ListBucketsRequest) (*ListBucketsResponse, error)
	HeadBucket(ctx context.Context, req HeadBucketRequest) error
	PutObject(ctx context.Context, req PutObjectRequest) (*PutObjectResponse, error)
	GetObject(ctx context.Context, req GetObjectRequest) (*GetObjectResponse, error)
	HeadObject(ctx context.Context, req HeadObjectRequest) (*HeadObjectResponse, error)
	DeleteObject(ctx context.Context, req DeleteObjectRequest) error
	ListObjects(ctx context.Context, req ListObjectsRequest) (*ListObjectsResponse, error)
	CreateMultipartUpload(ctx context.Context, req CreateMultipartUploadRequest) (*CreateMultipartUploadResponse, error)
	UploadPart(ctx context.Context, req UploadPartRequest) (*UploadPartResponse, error)
	CompleteMultipartUpload(ctx context.Context, req CompleteMultipartUploadRequest) (*CompleteMultipartUploadResponse, error)
	AbortMultipartUpload(ctx context.Context, req AbortMultipartUploadRequest) error
	Ready(ctx context.Context) error
}
