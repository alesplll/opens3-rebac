package tests

import (
	"context"
	"errors"
	"testing"

	userService "github.com/alesplll/opens3-rebac/services/users/internal/service/user"
	"github.com/alesplll/opens3-rebac/services/users/pkg/mocks"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/contextx/ipctx"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestUpdatePassword(t *testing.T) {
	type args struct {
		ctx             context.Context
		userID          string
		password        string
		passwordConfirm string
	}

	type userRepoMockBuilder func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock
	type txManagerMockBuilder func(t *testing.T, mc *minimock.Controller, args args, userRepoMock *mocks.UserRepositoryMock) *mocks.TxManagerMock

	ctxWithIP := context.WithValue(context.Background(), ipctx.IpKey, "127.0.0.1")
	ctxWithoutIP := context.Background()

	tests := []struct {
		name                string
		args                args
		wantErr             error
		wantValidationError bool
		buildRepoMock       userRepoMockBuilder
		buildTxManagerMock  txManagerMockBuilder
	}{
		{
			name: "success case",
			args: args{
				ctx:             ctxWithIP,
				userID:          "user-1",
				password:        "secret1",
				passwordConfirm: "secret1",
			},
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				mock := mocks.NewUserRepositoryMock(mc)
				mock.UpdatePasswordMock.Inspect(func(ctx context.Context, userID string, hashedPassword string) {
					require.Equal(t, args.userID, userID)
					require.NotEqual(t, args.password, hashedPassword)
					require.NoError(t, bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(args.password)))
				}).Return(nil)
				mock.LogPasswordMock.Expect(args.ctx, args.userID, "127.0.0.1").Return(nil)
				return mock
			},
			buildTxManagerMock: func(t *testing.T, mc *minimock.Controller, args args, userRepoMock *mocks.UserRepositoryMock) *mocks.TxManagerMock {
				mock := mocks.NewTxManagerMock(mc)
				mock.ReadCommittedMock.Set(func(ctx context.Context, fn db.Handler) error {
					return fn(ctx)
				})
				return mock
			},
		},
		{
			name: "repository update password error",
			args: args{
				ctx:             ctxWithIP,
				userID:          "user-2",
				password:        "secret1",
				passwordConfirm: "secret1",
			},
			wantErr: errors.New("update password failed"),
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				mock := mocks.NewUserRepositoryMock(mc)
				mock.UpdatePasswordMock.Return(errors.New("update password failed"))
				return mock
			},
			buildTxManagerMock: func(t *testing.T, mc *minimock.Controller, args args, userRepoMock *mocks.UserRepositoryMock) *mocks.TxManagerMock {
				mock := mocks.NewTxManagerMock(mc)
				mock.ReadCommittedMock.Set(func(ctx context.Context, fn db.Handler) error {
					return fn(ctx)
				})
				return mock
			},
		},
		{
			name: "repository log password error",
			args: args{
				ctx:             ctxWithIP,
				userID:          "user-3",
				password:        "secret1",
				passwordConfirm: "secret1",
			},
			wantErr: errors.New("log password failed"),
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				mock := mocks.NewUserRepositoryMock(mc)
				mock.UpdatePasswordMock.Return(nil)
				mock.LogPasswordMock.Expect(args.ctx, args.userID, "127.0.0.1").Return(errors.New("log password failed"))
				return mock
			},
			buildTxManagerMock: func(t *testing.T, mc *minimock.Controller, args args, userRepoMock *mocks.UserRepositoryMock) *mocks.TxManagerMock {
				mock := mocks.NewTxManagerMock(mc)
				mock.ReadCommittedMock.Set(func(ctx context.Context, fn db.Handler) error {
					return fn(ctx)
				})
				return mock
			},
		},
		{
			name: "transaction manager error",
			args: args{
				ctx:             ctxWithIP,
				userID:          "user-4",
				password:        "secret1",
				passwordConfirm: "secret1",
			},
			wantErr: errors.New("transaction failed"),
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				return mocks.NewUserRepositoryMock(mc)
			},
			buildTxManagerMock: func(t *testing.T, mc *minimock.Controller, args args, userRepoMock *mocks.UserRepositoryMock) *mocks.TxManagerMock {
				mock := mocks.NewTxManagerMock(mc)
				mock.ReadCommittedMock.Set(func(ctx context.Context, fn db.Handler) error {
					return errors.New("transaction failed")
				})
				return mock
			},
		},
		{
			name: "missing ip in context",
			args: args{
				ctx:             ctxWithoutIP,
				userID:          "user-5",
				password:        "secret1",
				passwordConfirm: "secret1",
			},
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				mock := mocks.NewUserRepositoryMock(mc)
				mock.UpdatePasswordMock.Return(nil)
				mock.LogPasswordMock.Expect(args.ctx, args.userID, "unknown").Return(nil)
				return mock
			},
			buildTxManagerMock: func(t *testing.T, mc *minimock.Controller, args args, userRepoMock *mocks.UserRepositoryMock) *mocks.TxManagerMock {
				mock := mocks.NewTxManagerMock(mc)
				mock.ReadCommittedMock.Set(func(ctx context.Context, fn db.Handler) error {
					return fn(ctx)
				})
				return mock
			},
		},
		{
			name: "empty password validation error",
			args: args{
				ctx:             ctxWithIP,
				userID:          "user-6",
				password:        "",
				passwordConfirm: "",
			},
			wantValidationError: true,
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				return mocks.NewUserRepositoryMock(mc)
			},
			buildTxManagerMock: func(t *testing.T, mc *minimock.Controller, args args, userRepoMock *mocks.UserRepositoryMock) *mocks.TxManagerMock {
				return mocks.NewTxManagerMock(mc)
			},
		},
		{
			name: "password mismatch validation error",
			args: args{
				ctx:             ctxWithIP,
				userID:          "user-7",
				password:        "secret1",
				passwordConfirm: "secret2",
			},
			wantValidationError: true,
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				return mocks.NewUserRepositoryMock(mc)
			},
			buildTxManagerMock: func(t *testing.T, mc *minimock.Controller, args args, userRepoMock *mocks.UserRepositoryMock) *mocks.TxManagerMock {
				return mocks.NewTxManagerMock(mc)
			},
		},
		{
			name: "password too short validation error",
			args: args{
				ctx:             ctxWithIP,
				userID:          "user-8",
				password:        "1234",
				passwordConfirm: "1234",
			},
			wantValidationError: true,
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				return mocks.NewUserRepositoryMock(mc)
			},
			buildTxManagerMock: func(t *testing.T, mc *minimock.Controller, args args, userRepoMock *mocks.UserRepositoryMock) *mocks.TxManagerMock {
				return mocks.NewTxManagerMock(mc)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			userRepoMock := tt.buildRepoMock(t, mc, tt.args)
			txManagerMock := tt.buildTxManagerMock(t, mc, tt.args, userRepoMock)

			service := userService.NewService(userRepoMock, txManagerMock)

			err := service.UpdatePassword(tt.args.ctx, tt.args.userID, tt.args.password, tt.args.passwordConfirm)

			if tt.wantValidationError {
				requireValidationError(t, err)
				require.Equal(t, uint64(0), txManagerMock.ReadCommittedBeforeCounter())
				require.Equal(t, uint64(0), userRepoMock.UpdatePasswordBeforeCounter())
				require.Equal(t, uint64(0), userRepoMock.LogPasswordBeforeCounter())
				return
			}

			require.Equal(t, tt.wantErr, err)
		})
	}
}
