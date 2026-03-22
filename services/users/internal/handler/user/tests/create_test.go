package tests

import (
	"context"
	"errors"
	"testing"

	userHandler "github.com/alesplll/opens3-rebac/services/users/internal/handler/user"
	"github.com/alesplll/opens3-rebac/services/users/internal/model"
	"github.com/alesplll/opens3-rebac/services/users/internal/service"
	"github.com/alesplll/opens3-rebac/services/users/pkg/mocks"
	desc "github.com/alesplll/opens3-rebac/shared/pkg/user/v1"
	"github.com/brianvoe/gofakeit"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestCreate(t *testing.T) {
	type userServiceMockFunc func(mc *minimock.Controller) service.UserService

	type args struct {
		ctx context.Context
		req *desc.CreateRequest
	}

	var (
		ctx = context.Background()
		mc  = minimock.NewController(t)

		id       = gofakeit.UUID()
		name     = gofakeit.Name()
		email    = gofakeit.Email()
		password = gofakeit.Password(true, true, true, true, true, 8)

		req = &desc.CreateRequest{
			UserInfo: &desc.UserInfo{
				Name:  name,
				Email: email,
			},

			Password:        password,
			PasswordConfirm: password,
		}

		info = model.UserInfo{
			Name:  name,
			Email: email,
		}
		serviceErr = errors.New("service error")

		res = &desc.CreateResponse{
			Id: id,
		}
	)

	tests := []struct {
		name            string
		args            args
		want            *desc.CreateResponse
		err             error
		userServiceMock userServiceMockFunc
	}{
		{
			name: "success case",
			args: args{
				ctx: ctx,
				req: req,
			},
			want: res,
			err:  nil,
			userServiceMock: func(mc *minimock.Controller) service.UserService {
				mock := mocks.NewUserServiceMock(mc)
				mock.CreateMock.Expect(ctx, info, password, password).Return(id, nil)
				return mock
			},
		},
		{
			name: "service error case",
			args: args{
				ctx: ctx,
				req: req,
			},
			want: nil,
			err:  serviceErr,
			userServiceMock: func(mc *minimock.Controller) service.UserService {
				mock := mocks.NewUserServiceMock(mc)
				mock.CreateMock.Expect(ctx, info, password, password).Return("", serviceErr)
				return mock
			},
		},
	}

	for _, tt := range tests {
		tt := tt // To avoid bugs in parralel tests
		t.Run(tt.name, func(t *testing.T) {
			userServiceMock := tt.userServiceMock(mc)
			handler := userHandler.NewHandler(userServiceMock)

			res, err := handler.Create(tt.args.ctx, tt.args.req)
			require.Equal(t, tt.err, err)
			require.Equal(t, tt.want, res)
		})
	}
}
