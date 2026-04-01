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

func TestDelete(t *testing.T) {
	type args struct {
		ctx    context.Context
		userID string
	}

	type userRepoMockBuilder func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock

	ctx := context.Background()

	tests := []struct {
		name          string
		args          args
		wantErr       error
		buildRepoMock userRepoMockBuilder
	}{
		{
			name: "success case",
			args: args{
				ctx:    ctx,
				userID: "user-1",
			},
			wantErr: nil,
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				mock := mocks.NewUserRepositoryMock(mc)
				mock.DeleteMock.Expect(args.ctx, args.userID).Return(nil)
				return mock
			},
		},
		{
			name: "repository error",
			args: args{
				ctx:    ctx,
				userID: "user-2",
			},
			wantErr: errors.New("repository delete failed"),
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				mock := mocks.NewUserRepositoryMock(mc)
				mock.DeleteMock.Expect(args.ctx, args.userID).Return(errors.New("repository delete failed"))
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

			err := service.Delete(tt.args.ctx, tt.args.userID)

			require.Equal(t, tt.wantErr, err)
		})
	}
}
