package gateway

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/config"
	domainerrors "github.com/alesplll/opens3-rebac/services/gateway/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/gateway/internal/service"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/tokens"
	"github.com/stretchr/testify/require"
)

type stubGatewayService struct {
	putObjectFn  func(ctx context.Context, req service.PutObjectRequest) (*service.PutObjectResponse, error)
	getObjectFn  func(ctx context.Context, req service.GetObjectRequest) (*service.GetObjectResponse, error)
	uploadPartFn func(ctx context.Context, req service.UploadPartRequest) (*service.UploadPartResponse, error)
	readyFn      func(ctx context.Context) error
}

type stubAuthService struct {
	loginFn               func(ctx context.Context, req service.LoginRequest) (*service.LoginResponse, error)
	refreshAccessTokenFn  func(ctx context.Context, req service.RefreshAccessTokenRequest) (*service.RefreshAccessTokenResponse, error)
	refreshRefreshTokenFn func(ctx context.Context, req service.RefreshRefreshTokenRequest) (*service.RefreshRefreshTokenResponse, error)
}

func (s *stubGatewayService) CreateBucket(context.Context, service.CreateBucketRequest) (*service.CreateBucketResponse, error) {
	panic("unexpected call")
}

func (s *stubGatewayService) DeleteBucket(context.Context, service.DeleteBucketRequest) error {
	panic("unexpected call")
}

func (s *stubGatewayService) ListBuckets(context.Context, service.ListBucketsRequest) (*service.ListBucketsResponse, error) {
	panic("unexpected call")
}

func (s *stubGatewayService) HeadBucket(context.Context, service.HeadBucketRequest) error {
	panic("unexpected call")
}

func (s *stubGatewayService) PutObject(ctx context.Context, req service.PutObjectRequest) (*service.PutObjectResponse, error) {
	if s.putObjectFn == nil {
		panic("unexpected call")
	}
	return s.putObjectFn(ctx, req)
}

func (s *stubGatewayService) GetObject(ctx context.Context, req service.GetObjectRequest) (*service.GetObjectResponse, error) {
	if s.getObjectFn == nil {
		panic("unexpected call")
	}
	return s.getObjectFn(ctx, req)
}

func (s *stubGatewayService) HeadObject(context.Context, service.HeadObjectRequest) (*service.HeadObjectResponse, error) {
	panic("unexpected call")
}

func (s *stubGatewayService) DeleteObject(context.Context, service.DeleteObjectRequest) error {
	panic("unexpected call")
}

func (s *stubGatewayService) ListObjects(context.Context, service.ListObjectsRequest) (*service.ListObjectsResponse, error) {
	panic("unexpected call")
}

func (s *stubGatewayService) CreateMultipartUpload(context.Context, service.CreateMultipartUploadRequest) (*service.CreateMultipartUploadResponse, error) {
	panic("unexpected call")
}

func (s *stubGatewayService) UploadPart(ctx context.Context, req service.UploadPartRequest) (*service.UploadPartResponse, error) {
	if s.uploadPartFn == nil {
		panic("unexpected call")
	}
	return s.uploadPartFn(ctx, req)
}

func (s *stubGatewayService) CompleteMultipartUpload(context.Context, service.CompleteMultipartUploadRequest) (*service.CompleteMultipartUploadResponse, error) {
	panic("unexpected call")
}

func (s *stubGatewayService) AbortMultipartUpload(context.Context, service.AbortMultipartUploadRequest) error {
	panic("unexpected call")
}

func (s *stubGatewayService) Ready(ctx context.Context) error {
	if s.readyFn == nil {
		return nil
	}
	return s.readyFn(ctx)
}

func (s *stubAuthService) Login(ctx context.Context, req service.LoginRequest) (*service.LoginResponse, error) {
	if s.loginFn == nil {
		panic("unexpected call")
	}
	return s.loginFn(ctx, req)
}

func (s *stubAuthService) RefreshAccessToken(ctx context.Context, req service.RefreshAccessTokenRequest) (*service.RefreshAccessTokenResponse, error) {
	if s.refreshAccessTokenFn == nil {
		panic("unexpected call")
	}
	return s.refreshAccessTokenFn(ctx, req)
}

func (s *stubAuthService) RefreshRefreshToken(ctx context.Context, req service.RefreshRefreshTokenRequest) (*service.RefreshRefreshTokenResponse, error) {
	if s.refreshRefreshTokenFn == nil {
		panic("unexpected call")
	}
	return s.refreshRefreshTokenFn(ctx, req)
}

type stubTokenVerifier struct {
	claims *tokens.UserClaims
	err    error
}

func (s *stubTokenVerifier) VerifyAccessToken(context.Context, string) (*tokens.UserClaims, error) {
	return s.claims, s.err
}

func (s *stubTokenVerifier) VerifyRefreshToken(context.Context, string) (*tokens.UserClaims, error) {
	return nil, errors.New("unexpected refresh token verification")
}

func TestPutObjectRejectsMissingContentLength(t *testing.T) {
	loadTestConfig(t)
	h := NewHandler(&stubGatewayService{}, &stubAuthService{}, 1024, &stubTokenVerifier{claims: &tokens.UserClaims{UserId: "user-1"}})

	req := httptest.NewRequest(http.MethodPut, "/bucket/key", strings.NewReader("payload"))
	req.Header.Set("Authorization", "Bearer token")
	req.ContentLength = -1

	rr := httptest.NewRecorder()
	h.putObject(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), "InvalidRequest")
	require.Contains(t, rr.Body.String(), "content length is required")
}

func TestUploadPartRejectsInvalidPartNumber(t *testing.T) {
	loadTestConfig(t)
	h := NewHandler(&stubGatewayService{}, &stubAuthService{}, 1024, &stubTokenVerifier{claims: &tokens.UserClaims{UserId: "user-1"}})

	req := httptest.NewRequest(http.MethodPut, "/bucket/key?uploadId=u1&partNumber=0", strings.NewReader("payload"))
	req.Header.Set("Authorization", "Bearer token")

	rr := httptest.NewRecorder()
	h.uploadPart(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), "invalid partNumber")
}

func TestGetObjectStreamsDirectlyToResponseWriter(t *testing.T) {
	serviceStub := &stubGatewayService{
		getObjectFn: func(ctx context.Context, req service.GetObjectRequest) (*service.GetObjectResponse, error) {
			_, err := io.WriteString(req.Writer, "streamed-data")
			require.NoError(t, err)
			return &service.GetObjectResponse{
				ContentType:   "application/octet-stream",
				ContentLength: int64(len("streamed-data")),
				ETag:          "\"etag\"",
				VersionID:     "v1",
				LastModified:  time.Unix(1700000000, 0).UTC(),
				TotalSize:     int64(len("streamed-data")),
			}, nil
		},
	}

	loadTestConfig(t)
	h := NewHandler(serviceStub, &stubAuthService{}, 1024, &stubTokenVerifier{claims: &tokens.UserClaims{UserId: "user-1"}})

	req := httptest.NewRequest(http.MethodGet, "/bucket/key", nil)
	req.Header.Set("Authorization", "Bearer token")

	rr := httptest.NewRecorder()
	h.getObject(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "streamed-data", rr.Body.String())
	require.Equal(t, "application/octet-stream", rr.Header().Get("Content-Type"))
	require.Equal(t, "13", rr.Header().Get("Content-Length"))
}

func TestReadyReturnsMappedError(t *testing.T) {
	loadTestConfig(t)
	h := NewHandler(&stubGatewayService{readyFn: func(context.Context) error {
		return domainerrors.ErrServiceUnavailable
	}}, &stubAuthService{}, 1024, &stubTokenVerifier{claims: &tokens.UserClaims{UserId: "user-1"}})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rr := httptest.NewRecorder()

	h.ready(rr, req)

	require.Equal(t, http.StatusServiceUnavailable, rr.Code)
	require.Contains(t, rr.Body.String(), "ServiceUnavailable")
}

func TestLoginEndpoint(t *testing.T) {
	loadTestConfig(t)
	h := NewHandler(&stubGatewayService{}, &stubAuthService{loginFn: func(ctx context.Context, req service.LoginRequest) (*service.LoginResponse, error) {
		require.Equal(t, "user@example.com", req.Email)
		require.Equal(t, "secret", req.Password)
		return &service.LoginResponse{RefreshToken: "refresh-token"}, nil
	}}, 1024, &stubTokenVerifier{})

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"user@example.com","password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.login(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.JSONEq(t, `{"refresh_token":"refresh-token"}`, rr.Body.String())
}

func TestRefreshAccessTokenEndpoint(t *testing.T) {
	loadTestConfig(t)
	h := NewHandler(&stubGatewayService{}, &stubAuthService{refreshAccessTokenFn: func(ctx context.Context, req service.RefreshAccessTokenRequest) (*service.RefreshAccessTokenResponse, error) {
		require.Equal(t, "refresh-token", req.RefreshToken)
		return &service.RefreshAccessTokenResponse{AccessToken: "access-token"}, nil
	}}, 1024, &stubTokenVerifier{})

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh/access", strings.NewReader(`{"refresh_token":"refresh-token"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.refreshAccessToken(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.JSONEq(t, `{"access_token":"access-token"}`, rr.Body.String())
}

func TestRefreshRefreshTokenEndpoint(t *testing.T) {
	loadTestConfig(t)
	h := NewHandler(&stubGatewayService{}, &stubAuthService{refreshRefreshTokenFn: func(ctx context.Context, req service.RefreshRefreshTokenRequest) (*service.RefreshRefreshTokenResponse, error) {
		require.Equal(t, "refresh-token", req.RefreshToken)
		return &service.RefreshRefreshTokenResponse{RefreshToken: "new-refresh-token"}, nil
	}}, 1024, &stubTokenVerifier{})

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh/refresh", strings.NewReader(`{"refresh_token":"refresh-token"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.refreshRefreshToken(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.JSONEq(t, `{"refresh_token":"new-refresh-token"}`, rr.Body.String())
}

func loadTestConfig(t *testing.T) {
	t.Helper()
	t.Setenv("LOGGER_LEVEL", "info")
	t.Setenv("LOGGER_AS_JSON", "false")
	t.Setenv("LOGGER_ENABLE_OLTP", "false")
	t.Setenv("OTEL_SERVICE_NAME", "gateway-test")
	t.Setenv("OTEL_SERVICE_VERSION", "test")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	t.Setenv("OTEL_ENVIRONMENT", "test")
	t.Setenv("OTEL_METRICS_PUSH_TIMEOUT", "1s")
	t.Setenv("JWT_SECRET", "test-secret")
	require.NoError(t, config.Load())
}
