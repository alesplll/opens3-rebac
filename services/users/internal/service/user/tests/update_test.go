package tests

import (
	"context"
	"errors"
	"testing"

	userService "github.com/alesplll/opens3-rebac/services/users/internal/service/user"
	"github.com/alesplll/opens3-rebac/services/users/pkg/mocks"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestUpdate(t *testing.T) {
	type args struct {
		ctx    context.Context
		userID string
		name   *string
		email  *string
	}

	type userRepoMockBuilder func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock

	ctx := context.Background()

	tests := []struct {
		name                string
		args                args
		wantErr             error
		wantValidationError bool
		buildRepoMock       userRepoMockBuilder
	}{
		{
			name: "success case",
			args: args{
				ctx:    ctx,
				userID: "user-1",
				name:   strPtr("John Doe"),
				email:  strPtr("john@example.com"),
			},
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				mock := mocks.NewUserRepositoryMock(mc)
				mock.UpdateMock.Expect(args.ctx, args.userID, args.name, args.email).Return(nil)
				return mock
			},
		},
		{
			name: "repository error",
			args: args{
				ctx:    ctx,
				userID: "user-2",
				name:   strPtr("John Doe"),
				email:  strPtr("john@example.com"),
			},
			wantErr: errors.New("repository update failed"),
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				mock := mocks.NewUserRepositoryMock(mc)
				mock.UpdateMock.Expect(args.ctx, args.userID, args.name, args.email).Return(errors.New("repository update failed"))
				return mock
			},
		},
		{
			name: "empty name validation error",
			args: args{
				ctx:    ctx,
				userID: "user-3",
				name:   strPtr(""),
				email:  strPtr("john@example.com"),
			},
			wantValidationError: true,
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				return mocks.NewUserRepositoryMock(mc)
			},
		},
		{
			name: "empty email validation error",
			args: args{
				ctx:    ctx,
				userID: "user-4",
				name:   strPtr("John Doe"),
				email:  strPtr(""),
			},
			wantValidationError: true,
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				return mocks.NewUserRepositoryMock(mc)
			},
		},
		{
			name: "invalid email validation error",
			args: args{
				ctx:    ctx,
				userID: "user-5",
				name:   strPtr("John Doe"),
				email:  strPtr("invalid-email"),
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

			err := service.Update(tt.args.ctx, tt.args.userID, tt.args.name, tt.args.email)

			if tt.wantValidationError {
				requireValidationError(t, err)
				require.Equal(t, uint64(0), userRepoMock.UpdateBeforeCounter())
				return
			}

			require.Equal(t, tt.wantErr, err)
		})
	}
}
