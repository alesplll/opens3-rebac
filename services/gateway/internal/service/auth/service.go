package auth

import (
	"context"
	"errors"
	"strings"

	grpcclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc"
	domainerrors "github.com/alesplll/opens3-rebac/services/gateway/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/gateway/internal/service"
	authv1 "github.com/alesplll/opens3-rebac/shared/pkg/go/auth/v1"
	userv1 "github.com/alesplll/opens3-rebac/shared/pkg/go/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authService struct {
	authClient  grpcclient.AuthClient
	usersClient grpcclient.UsersClient
}

func NewService(authClient grpcclient.AuthClient, usersClient grpcclient.UsersClient) service.AuthService {
	return &authService{
		authClient:  authClient,
		usersClient: usersClient,
	}
}

func (s *authService) Login(ctx context.Context, req service.LoginRequest) (*service.LoginResponse, error) {
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
		return nil, domainerrors.ErrUnauthorized
	}

	credentialsResp, err := s.usersClient.ValidateCredentials(ctx, &userv1.ValidateCredentialsRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return nil, mapAuthError(err)
	}
	if credentialsResp == nil || !credentialsResp.GetValid() || strings.TrimSpace(credentialsResp.GetUserId()) == "" {
		return nil, domainerrors.ErrUnauthorized
	}

	loginResp, err := s.authClient.Login(ctx, &authv1.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return nil, mapAuthError(err)
	}

	return &service.LoginResponse{RefreshToken: loginResp.GetRefreshToken()}, nil
}

func (s *authService) RefreshAccessToken(ctx context.Context, req service.RefreshAccessTokenRequest) (*service.RefreshAccessTokenResponse, error) {
	if strings.TrimSpace(req.RefreshToken) == "" {
		return nil, domainerrors.ErrUnauthorized
	}

	resp, err := s.authClient.GetAccessToken(ctx, &authv1.GetAccessTokenRequest{RefreshToken: req.RefreshToken})
	if err != nil {
		return nil, mapAuthError(err)
	}

	return &service.RefreshAccessTokenResponse{AccessToken: resp.GetAccessToken()}, nil
}

func (s *authService) RefreshRefreshToken(ctx context.Context, req service.RefreshRefreshTokenRequest) (*service.RefreshRefreshTokenResponse, error) {
	if strings.TrimSpace(req.RefreshToken) == "" {
		return nil, domainerrors.ErrUnauthorized
	}

	resp, err := s.authClient.GetRefreshToken(ctx, &authv1.GetRefreshTokenRequest{RefreshToken: req.RefreshToken})
	if err != nil {
		return nil, mapAuthError(err)
	}

	return &service.RefreshRefreshTokenResponse{RefreshToken: resp.GetRefreshToken()}, nil
}

func mapAuthError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return domainerrors.ErrServiceUnavailable
		}
		return err
	}

	switch st.Code() {
	case codes.Unauthenticated, codes.PermissionDenied, codes.InvalidArgument, codes.NotFound:
		return domainerrors.ErrUnauthorized
	case codes.Unavailable, codes.DeadlineExceeded:
		return domainerrors.ErrServiceUnavailable
	default:
		return err
	}
}
