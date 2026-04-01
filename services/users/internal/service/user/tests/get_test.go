package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alesplll/opens3-rebac/services/users/internal/model"
	userService "github.com/alesplll/opens3-rebac/services/users/internal/service/user"
	"github.com/alesplll/opens3-rebac/services/users/pkg/mocks"
	"github.com/brianvoe/gofakeit"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	type args struct {
		ctx context.Context
		id  string
	}

	type userRepoMockBuilder func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock

	ctx := context.Background()

	tests := []struct {
		name          string
		args          args
		wantUser      *model.User
		wantErr       error
		buildRepoMock userRepoMockBuilder
	}{
		{
			name: "success case",
			args: args{
				ctx: ctx,
				id:  gofakeit.UUID(),
			},
			wantErr: nil,
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				mock := mocks.NewUserRepositoryMock(mc)
				mock.GetMock.Expect(args.ctx, args.id).Return(&model.User{
					Id: "c8f8d2bf-f5ac-4cf7-a3bd-f7a7301f6fc2",
					UserInfo: model.UserInfo{
						Name:  "John Doe",
						Email: "john@example.com",
					},
					CreatedAt: time.Date(2026, time.April, 2, 10, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2026, time.April, 2, 11, 0, 0, 0, time.UTC),
				}, nil)
				return mock
			},
			wantUser: &model.User{
				Id: "c8f8d2bf-f5ac-4cf7-a3bd-f7a7301f6fc2",
				UserInfo: model.UserInfo{
					Name:  "John Doe",
					Email: "john@example.com",
				},
				CreatedAt: time.Date(2026, time.April, 2, 10, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, time.April, 2, 11, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "repository error",
			args: args{
				ctx: ctx,
				id:  gofakeit.UUID(),
			},
			wantUser: nil,
			wantErr:  errors.New("repository get failed"),
			buildRepoMock: func(t *testing.T, mc *minimock.Controller, args args) *mocks.UserRepositoryMock {
				mock := mocks.NewUserRepositoryMock(mc)
				mock.GetMock.Expect(args.ctx, args.id).Return(nil, errors.New("repository get failed"))
				return mock
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			userRepoMock := tt.buildRepoMock(t, mc, tt.args)
			txManagerMock := mocks.NewTxManagerMock(mc)

			service := userService.NewService(userRepoMock, txManagerMock)

			user, err := service.Get(tt.args.ctx, tt.args.id)

			require.Equal(t, tt.wantUser, user)
			require.Equal(t, tt.wantErr, err)
		})
	}
}
