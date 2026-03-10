package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	userHandler "github.com/alesplll/opens3-rebac/services/users/internal/handler/user"
	"github.com/alesplll/opens3-rebac/services/users/internal/model"
	"github.com/alesplll/opens3-rebac/services/users/internal/service"
	"github.com/alesplll/opens3-rebac/services/users/pkg/mocks"
	desc "github.com/alesplll/opens3-rebac/shared/pkg/user/v1"
	"github.com/brianvoe/gofakeit"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGet(t *testing.T) {
	type userServiceMockFunc func(mc *minimock.Controller) service.UserService

	type args struct {
		ctx context.Context
		req *desc.GetRequest
	}

	var (
		ctx = context.Background()
		mc  = minimock.NewController(t)

		id    = gofakeit.UUID()
		name  = gofakeit.Name()
		email = gofakeit.Email()

		createdAt = time.Now()
		updatedAt = time.Now()

		req = &desc.GetRequest{
			Id: id,
		}

		user = &model.User{
			Id: id,
			UserInfo: model.UserInfo{
				Name:  name,
				Email: email,
			},
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}

		serviceErr = errors.New("service error")

		res = &desc.GetResponse{
			User: &desc.User{
				Id: id,
				UserInfo: &desc.UserInfo{
					Name:  name,
					Email: email,
				},
				CreatedAt: timestamppb.New(createdAt),
				UpdatedAt: timestamppb.New(updatedAt),
			},
		}
	)

	tests := []struct {
		name            string
		args            args
		want            *desc.GetResponse
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
				mock.GetMock.Expect(ctx, id).Return(user, nil)
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
				mock.GetMock.Expect(ctx, id).Return(nil, serviceErr)
				return mock
			},
		},
	}

	for _, tt := range tests {
		tt := tt // To avoid bugs in parralel tests
		t.Run(tt.name, func(t *testing.T) {
			userServiceMock := tt.userServiceMock(mc)
			handler := userHandler.NewHandler(userServiceMock)

			res, err := handler.Get(tt.args.ctx, tt.args.req)
			require.Equal(t, tt.err, err)
			require.Equal(t, tt.want, res)
		})
	}
}
