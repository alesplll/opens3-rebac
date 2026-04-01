package tests

import (
	"context"
	"errors"
	"testing"

	domainerrors "github.com/alesplll/opens3-rebac/services/users/internal/errors/domain_errors"
	userService "github.com/alesplll/opens3-rebac/services/users/internal/service/user"
	"github.com/alesplll/opens3-rebac/services/users/pkg/mocks"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestValidateCredentials(t *testing.T) {
	type args struct {
		ctx      context.Context
		email    string
		password string
	}

	type userRepoMockBuilder func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock

	logger.SetNopLogger()

	ctx := context.Background()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("secret1"), bcrypt.DefaultCost)
	require.NoError(t, err)

	tests := []struct {
		name          string
		args          args
		wantValid     bool
		wantUserID    string
		buildRepoMock userRepoMockBuilder
	}{
		{
			name: "success case",
			args: args{
				ctx:      ctx,
				email:    "john@example.com",
				password: "secret1",
			},
			wantValid:  true,
			wantUserID: "user-1",
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				mock := mocks.NewUserRepositoryMock(mc)
				mock.GetUserCredentialsMock.Set(func(ctx context.Context, email string) (string, string, error) {
					require.Equal(t, args.email, email)
					return "user-1", string(hashedPassword), nil
				})
				return mock
			},
		},
		{
			name: "user not found",
			args: args{
				ctx:      ctx,
				email:    "missing@example.com",
				password: "secret1",
			},
			wantValid:  false,
			wantUserID: "",
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				mock := mocks.NewUserRepositoryMock(mc)
				mock.GetUserCredentialsMock.Set(func(ctx context.Context, email string) (string, string, error) {
					require.Equal(t, args.email, email)
					return "", "", domainerrors.ErrUserNotFound
				})
				return mock
			},
		},
		{
			name: "repository error",
			args: args{
				ctx:      ctx,
				email:    "john@example.com",
				password: "secret1",
			},
			wantValid:  false,
			wantUserID: "",
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				mock := mocks.NewUserRepositoryMock(mc)
				mock.GetUserCredentialsMock.Set(func(ctx context.Context, email string) (string, string, error) {
					require.Equal(t, args.email, email)
					return "", "", errors.New("repository failed")
				})
				return mock
			},
		},
		{
			name: "invalid password",
			args: args{
				ctx:      ctx,
				email:    "john@example.com",
				password: "wrong-secret",
			},
			wantValid:  false,
			wantUserID: "",
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				mock := mocks.NewUserRepositoryMock(mc)
				mock.GetUserCredentialsMock.Set(func(ctx context.Context, email string) (string, string, error) {
					require.Equal(t, args.email, email)
					return "user-1", string(hashedPassword), nil
				})
				return mock
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			userRepoMock := tt.buildRepoMock(t, mc, tt.args)
			txManagerMock := mocks.NewTxManagerMock(mc)

			service := userService.NewService(userRepoMock, txManagerMock)

			valid, userID := service.ValidateCredentials(tt.args.ctx, tt.args.email, tt.args.password)

			require.Equal(t, tt.wantValid, valid)
			require.Equal(t, tt.wantUserID, userID)
		})
	}
}
