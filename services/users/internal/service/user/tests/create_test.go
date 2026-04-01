package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alesplll/opens3-rebac/services/users/internal/model"
	userService "github.com/alesplll/opens3-rebac/services/users/internal/service/user"
	"github.com/alesplll/opens3-rebac/services/users/pkg/mocks"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestCreate(t *testing.T) {
	type args struct {
		ctx             context.Context
		userInfo        model.UserInfo
		password        string
		passwordConfirm string
	}

	type userRepoMockBuilder func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock

	logger.SetNopLogger()

	ctx := context.Background()

	tests := []struct {
		name                string
		args                args
		wantID              string
		wantErr             error
		wantValidationError bool
		buildRepoMock       userRepoMockBuilder
	}{
		{
			name: "success case",
			args: args{
				ctx: ctx,
				userInfo: model.UserInfo{
					Name:  "John Doe",
					Email: "john@example.com",
				},
				password:        "secret1",
				passwordConfirm: "secret1",
			},
			wantID: "user-1",
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				mock := mocks.NewUserRepositoryMock(mc)
				mock.CreateMock.Inspect(func(ctx context.Context, userInfo *model.UserInfo, hashedPassword string, createdAt time.Time) {
					require.Equal(t, &args.userInfo, userInfo)
					require.NotZero(t, createdAt)
					require.NotEqual(t, args.password, hashedPassword)
					require.NoError(t, bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(args.password)))
				}).Return("user-1", nil)
				return mock
			},
		},
		{
			name: "repository error",
			args: args{
				ctx: ctx,
				userInfo: model.UserInfo{
					Name:  "John Doe",
					Email: "john@example.com",
				},
				password:        "secret1",
				passwordConfirm: "secret1",
			},
			wantErr: errors.New("repository create failed"),
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				mock := mocks.NewUserRepositoryMock(mc)
				mock.CreateMock.Return("", errors.New("repository create failed"))
				return mock
			},
		},
		{
			name: "empty name validation error",
			args: args{
				ctx: ctx,
				userInfo: model.UserInfo{
					Name:  "",
					Email: "john@example.com",
				},
				password:        "secret1",
				passwordConfirm: "secret1",
			},
			wantValidationError: true,
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				return mocks.NewUserRepositoryMock(mc)
			},
		},
		{
			name: "empty email validation error",
			args: args{
				ctx: ctx,
				userInfo: model.UserInfo{
					Name:  "John Doe",
					Email: "",
				},
				password:        "secret1",
				passwordConfirm: "secret1",
			},
			wantValidationError: true,
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				return mocks.NewUserRepositoryMock(mc)
			},
		},
		{
			name: "invalid email validation error",
			args: args{
				ctx: ctx,
				userInfo: model.UserInfo{
					Name:  "John Doe",
					Email: "invalid-email",
				},
				password:        "secret1",
				passwordConfirm: "secret1",
			},
			wantValidationError: true,
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				return mocks.NewUserRepositoryMock(mc)
			},
		},
		{
			name: "empty password validation error",
			args: args{
				ctx: ctx,
				userInfo: model.UserInfo{
					Name:  "John Doe",
					Email: "john@example.com",
				},
				password:        "",
				passwordConfirm: "",
			},
			wantValidationError: true,
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				return mocks.NewUserRepositoryMock(mc)
			},
		},
		{
			name: "password mismatch validation error",
			args: args{
				ctx: ctx,
				userInfo: model.UserInfo{
					Name:  "John Doe",
					Email: "john@example.com",
				},
				password:        "secret1",
				passwordConfirm: "secret2",
			},
			wantValidationError: true,
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				return mocks.NewUserRepositoryMock(mc)
			},
		},
		{
			name: "password too short validation error",
			args: args{
				ctx: ctx,
				userInfo: model.UserInfo{
					Name:  "John Doe",
					Email: "john@example.com",
				},
				password:        "1234",
				passwordConfirm: "1234",
			},
			wantValidationError: true,
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				return mocks.NewUserRepositoryMock(mc)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			userRepoMock := tt.buildRepoMock(t, mc, tt.args)
			txManagerMock := mocks.NewTxManagerMock(mc)

			service := userService.NewService(userRepoMock, txManagerMock)

			id, err := service.Create(tt.args.ctx, tt.args.userInfo, tt.args.password, tt.args.passwordConfirm)

			if tt.wantValidationError {
				requireValidationError(t, err)
				require.Equal(t, uint64(0), userRepoMock.CreateBeforeCounter())
				return
			}

			require.Equal(t, tt.wantID, id)
			require.Equal(t, tt.wantErr, err)
		})
	}
}
