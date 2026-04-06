package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/alesplll/opens3-rebac/services/auth/internal/config"
	domainerrors "github.com/alesplll/opens3-rebac/services/auth/internal/errors/domain"
	"github.com/alesplll/opens3-rebac/services/auth/internal/model"
	authService "github.com/alesplll/opens3-rebac/services/auth/internal/service/auth"
	"github.com/alesplll/opens3-rebac/services/auth/pkg/mocks"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"github.com/brianvoe/gofakeit"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestLogin(t *testing.T) {
	type args struct {
		ctx      context.Context
		email    string
		password string
	}

	type mockBuilders struct {
		buildUserClientMock func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserClientMock
		buildRepoMock       func(t *testing.T, mc *minimock.Controller, args args) *mocks.AuthRepositoryMock
		buildTokenMock      func(t *testing.T, mc *minimock.Controller) *mocks.TokenServiceMock
	}

	logger.SetNopLogger()

	// Set required env vars for config.Load()
	t.Setenv("LOGGER_LEVEL", "info")
	t.Setenv("LOGGER_AS_JSON", "false")
	t.Setenv("LOGGER_ENABLE_OLTP", "false")
	t.Setenv("OTEL_SERVICE_NAME", "auth-test")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	t.Setenv("OTEL_ENVIRONMENT", "test")
	t.Setenv("OTEL_SERVICE_VERSION", "0.0.1")
	t.Setenv("OTEL_METRICS_PUSH_TIMEOUT", "5s")
	t.Setenv("GRPC_HOST", "localhost")
	t.Setenv("GRPC_PORT", "50051")
	t.Setenv("USER_SERVER_GRPC_HOST", "localhost")
	t.Setenv("USER_SERVER_GRPC_PORT", "50054")
	t.Setenv("CACHE_HOST", "localhost")
	t.Setenv("INTERNAL_CACHE_PORT", "6379")
	t.Setenv("EXTERNAL_CACHE_PORT", "6379")
	t.Setenv("REFRESH_TOKEN_SECRET", "test-secret")
	t.Setenv("ACCESS_TOKEN_SECRET", "test-secret")
	require.NoError(t, config.Load())

	ctx := context.Background()
	email := gofakeit.Email()
	password := gofakeit.Password(true, true, true, true, true, 8)
	refreshToken := gofakeit.UUID()
	userID := gofakeit.UUID()

	repoErr := errors.New("repo error")
	userClientErr := errors.New("user client error")
	tokenErr := errors.New("token generation error")

	tests := []struct {
		name                string
		args                args
		wantToken           string
		wantErr             error
		wantValidationError bool
		mocks               mockBuilders
	}{
		{
			name:      "success case",
			args:      args{ctx: ctx, email: email, password: password},
			wantToken: refreshToken,
			mocks: mockBuilders{
				buildUserClientMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserClientMock {
					mock := mocks.NewUserClientMock(mc)
					mock.ValidateCredentialsMock.Inspect(func(_ context.Context, e, p string) {
						require.Equal(t, args.email, e)
						require.Equal(t, args.password, p)
					}).Return(model.ValidateCredentialsResult{Valid: true, UserID: userID}, nil)
					return mock
				},
				buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.AuthRepositoryMock {
					mock := mocks.NewAuthRepositoryMock(mc)
					mock.GetLoginAttemptsMock.Inspect(func(_ context.Context, e string) {
						require.Equal(t, args.email, e)
					}).Return(0, nil)
					mock.ResetLoginAttemptsMock.Inspect(func(_ context.Context, e string) {
						require.Equal(t, args.email, e)
					}).Return(nil)
					return mock
				},
				buildTokenMock: func(t *testing.T, mc *minimock.Controller) *mocks.TokenServiceMock {
					mock := mocks.NewTokenServiceMock(mc)
					mock.GenerateRefreshTokenMock.Return(refreshToken, nil)
					return mock
				},
			},
		},
		{
			name:                "empty email validation error",
			args:                args{ctx: ctx, email: "", password: password},
			wantValidationError: true,
			mocks: mockBuilders{
				buildUserClientMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserClientMock {
					return mocks.NewUserClientMock(mc)
				},
				buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.AuthRepositoryMock {
					return mocks.NewAuthRepositoryMock(mc)
				},
				buildTokenMock: func(t *testing.T, mc *minimock.Controller) *mocks.TokenServiceMock {
					return mocks.NewTokenServiceMock(mc)
				},
			},
		},
		{
			name:                "empty password validation error",
			args:                args{ctx: ctx, email: email, password: ""},
			wantValidationError: true,
			mocks: mockBuilders{
				buildUserClientMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserClientMock {
					return mocks.NewUserClientMock(mc)
				},
				buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.AuthRepositoryMock {
					return mocks.NewAuthRepositoryMock(mc)
				},
				buildTokenMock: func(t *testing.T, mc *minimock.Controller) *mocks.TokenServiceMock {
					return mocks.NewTokenServiceMock(mc)
				},
			},
		},
		{
			name:    "too many login attempts",
			args:    args{ctx: ctx, email: email, password: password},
			wantErr: domainerrors.ErrTooManyAttempts,
			mocks: mockBuilders{
				buildUserClientMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserClientMock {
					return mocks.NewUserClientMock(mc)
				},
				buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.AuthRepositoryMock {
					mock := mocks.NewAuthRepositoryMock(mc)
					mock.GetLoginAttemptsMock.Inspect(func(_ context.Context, e string) {
						require.Equal(t, args.email, e)
					}).Return(config.AppConfig().Security.MaxLoginAttempts(), nil)
					return mock
				},
				buildTokenMock: func(t *testing.T, mc *minimock.Controller) *mocks.TokenServiceMock {
					return mocks.NewTokenServiceMock(mc)
				},
			},
		},
		{
			name:    "get login attempts repo error",
			args:    args{ctx: ctx, email: email, password: password},
			wantErr: repoErr,
			mocks: mockBuilders{
				buildUserClientMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserClientMock {
					return mocks.NewUserClientMock(mc)
				},
				buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.AuthRepositoryMock {
					mock := mocks.NewAuthRepositoryMock(mc)
					mock.GetLoginAttemptsMock.Return(0, repoErr)
					return mock
				},
				buildTokenMock: func(t *testing.T, mc *minimock.Controller) *mocks.TokenServiceMock {
					return mocks.NewTokenServiceMock(mc)
				},
			},
		},
		{
			name:    "invalid credentials",
			args:    args{ctx: ctx, email: email, password: password},
			wantErr: domainerrors.ErrInvalidEmailOrPassword,
			mocks: mockBuilders{
				buildUserClientMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserClientMock {
					mock := mocks.NewUserClientMock(mc)
					mock.ValidateCredentialsMock.Return(model.ValidateCredentialsResult{Valid: false}, nil)
					return mock
				},
				buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.AuthRepositoryMock {
					mock := mocks.NewAuthRepositoryMock(mc)
					mock.GetLoginAttemptsMock.Return(0, nil)
					mock.IncrementLoginAttemptsMock.Return(1, nil)
					return mock
				},
				buildTokenMock: func(t *testing.T, mc *minimock.Controller) *mocks.TokenServiceMock {
					return mocks.NewTokenServiceMock(mc)
				},
			},
		},
		{
			name:    "user client error",
			args:    args{ctx: ctx, email: email, password: password},
			wantErr: userClientErr,
			mocks: mockBuilders{
				buildUserClientMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserClientMock {
					mock := mocks.NewUserClientMock(mc)
					mock.ValidateCredentialsMock.Return(model.ValidateCredentialsResult{}, userClientErr)
					return mock
				},
				buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.AuthRepositoryMock {
					mock := mocks.NewAuthRepositoryMock(mc)
					mock.GetLoginAttemptsMock.Return(0, nil)
					mock.IncrementLoginAttemptsMock.Return(1, nil)
					return mock
				},
				buildTokenMock: func(t *testing.T, mc *minimock.Controller) *mocks.TokenServiceMock {
					return mocks.NewTokenServiceMock(mc)
				},
			},
		},
		{
			name:    "token generation error",
			args:    args{ctx: ctx, email: email, password: password},
			wantErr: tokenErr,
			mocks: mockBuilders{
				buildUserClientMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserClientMock {
					mock := mocks.NewUserClientMock(mc)
					mock.ValidateCredentialsMock.Return(model.ValidateCredentialsResult{Valid: true, UserID: userID}, nil)
					return mock
				},
				buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.AuthRepositoryMock {
					mock := mocks.NewAuthRepositoryMock(mc)
					mock.GetLoginAttemptsMock.Return(0, nil)
					mock.ResetLoginAttemptsMock.Return(nil)
					return mock
				},
				buildTokenMock: func(t *testing.T, mc *minimock.Controller) *mocks.TokenServiceMock {
					mock := mocks.NewTokenServiceMock(mc)
					mock.GenerateRefreshTokenMock.Return("", tokenErr)
					return mock
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			userClientMock := tt.mocks.buildUserClientMock(t, mc, tt.args)
			repoMock := tt.mocks.buildRepoMock(t, mc, tt.args)
			tokenMock := tt.mocks.buildTokenMock(t, mc)

			svc := authService.NewService(userClientMock, tokenMock, repoMock)

			token, err := svc.Login(tt.args.ctx, tt.args.email, tt.args.password)

			if tt.wantValidationError {
				requireValidationError(t, err)
				require.Equal(t, uint64(0), userClientMock.ValidateCredentialsBeforeCounter())
				return
			}

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Empty(t, token)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantToken, token)
		})
	}
}
