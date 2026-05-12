package auth

import (
	"context"
	"testing"

	grpcclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc"
	domainerrors "github.com/alesplll/opens3-rebac/services/gateway/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/gateway/internal/service"
	authv1 "github.com/alesplll/opens3-rebac/shared/pkg/go/auth/v1"
	userv1 "github.com/alesplll/opens3-rebac/shared/pkg/go/user/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

type stubAuthClient struct {
	loginFn           func(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error)
	getRefreshTokenFn func(ctx context.Context, req *authv1.GetRefreshTokenRequest) (*authv1.GetRefreshTokenResponse, error)
	getAccessTokenFn  func(ctx context.Context, req *authv1.GetAccessTokenRequest) (*authv1.GetAccessTokenResponse, error)
}

func (s *stubAuthClient) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	if s.loginFn == nil {
		panic("unexpected call")
	}
	return s.loginFn(ctx, req)
}

func (s *stubAuthClient) GetRefreshToken(ctx context.Context, req *authv1.GetRefreshTokenRequest) (*authv1.GetRefreshTokenResponse, error) {
	if s.getRefreshTokenFn == nil {
		panic("unexpected call")
	}
	return s.getRefreshTokenFn(ctx, req)
}

func (s *stubAuthClient) GetAccessToken(ctx context.Context, req *authv1.GetAccessTokenRequest) (*authv1.GetAccessTokenResponse, error) {
	if s.getAccessTokenFn == nil {
		panic("unexpected call")
	}
	return s.getAccessTokenFn(ctx, req)
}

func (s *stubAuthClient) ValidateToken(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	panic("unexpected call")
}

type stubUsersClient struct {
	validateCredentialsFn func(ctx context.Context, req *userv1.ValidateCredentialsRequest) (*userv1.ValidateCredentialsResponse, error)
}

func (s *stubUsersClient) ValidateCredentials(ctx context.Context, req *userv1.ValidateCredentialsRequest) (*userv1.ValidateCredentialsResponse, error) {
	if s.validateCredentialsFn == nil {
		panic("unexpected call")
	}
	return s.validateCredentialsFn(ctx, req)
}

var _ grpcclient.AuthClient = (*stubAuthClient)(nil)
var _ grpcclient.UsersClient = (*stubUsersClient)(nil)

func TestLogin(t *testing.T) {
	svc := NewService(
		&stubAuthClient{loginFn: func(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
			require.Equal(t, "user@example.com", req.GetEmail())
			require.Equal(t, "secret", req.GetPassword())
			return &authv1.LoginResponse{RefreshToken: "refresh-token"}, nil
		}},
		&stubUsersClient{validateCredentialsFn: func(ctx context.Context, req *userv1.ValidateCredentialsRequest) (*userv1.ValidateCredentialsResponse, error) {
			require.Equal(t, "user@example.com", req.GetEmail())
			require.Equal(t, "secret", req.GetPassword())
			return &userv1.ValidateCredentialsResponse{Valid: true, UserId: "user-1"}, nil
		}},
	)

	resp, err := svc.Login(context.Background(), service.LoginRequest{Email: "user@example.com", Password: "secret"})
	require.NoError(t, err)
	require.Equal(t, "refresh-token", resp.RefreshToken)
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	svc := NewService(
		&stubAuthClient{},
		&stubUsersClient{validateCredentialsFn: func(ctx context.Context, req *userv1.ValidateCredentialsRequest) (*userv1.ValidateCredentialsResponse, error) {
			return &userv1.ValidateCredentialsResponse{Valid: false}, nil
		}},
	)

	resp, err := svc.Login(context.Background(), service.LoginRequest{Email: "user@example.com", Password: "bad"})
	require.Nil(t, resp)
	require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
}

func TestRefreshAccessToken(t *testing.T) {
	svc := NewService(
		&stubAuthClient{getAccessTokenFn: func(ctx context.Context, req *authv1.GetAccessTokenRequest) (*authv1.GetAccessTokenResponse, error) {
			require.Equal(t, "refresh-token", req.GetRefreshToken())
			return &authv1.GetAccessTokenResponse{AccessToken: "access-token"}, nil
		}},
		&stubUsersClient{},
	)

	resp, err := svc.RefreshAccessToken(context.Background(), service.RefreshAccessTokenRequest{RefreshToken: "refresh-token"})
	require.NoError(t, err)
	require.Equal(t, "access-token", resp.AccessToken)
}

func TestRefreshRefreshToken(t *testing.T) {
	svc := NewService(
		&stubAuthClient{getRefreshTokenFn: func(ctx context.Context, req *authv1.GetRefreshTokenRequest) (*authv1.GetRefreshTokenResponse, error) {
			require.Equal(t, "refresh-token", req.GetRefreshToken())
			return &authv1.GetRefreshTokenResponse{RefreshToken: "new-refresh-token"}, nil
		}},
		&stubUsersClient{},
	)

	resp, err := svc.RefreshRefreshToken(context.Background(), service.RefreshRefreshTokenRequest{RefreshToken: "refresh-token"})
	require.NoError(t, err)
	require.Equal(t, "new-refresh-token", resp.RefreshToken)
}
