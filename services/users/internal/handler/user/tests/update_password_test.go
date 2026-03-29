package tests

import (
	"context"
	"errors"
	"testing"

	userHandler "github.com/alesplll/opens3-rebac/services/users/internal/handler/user"
	"github.com/alesplll/opens3-rebac/services/users/internal/service"
	"github.com/alesplll/opens3-rebac/services/users/pkg/mocks"
	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/user/v1"
	"github.com/brianvoe/gofakeit"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestUpdatePassword(t *testing.T) {
	type userServiceMockFunc func(mc *minimock.Controller) service.UserService

	type args struct {
		req *desc.UpdatePasswordRequest
	}

	var (
		ctx = context.Background()
		mc  = minimock.NewController(t)

		id = gofakeit.UUID()

		password = gofakeit.Password(true, true, true, true, true, 8)

		req = &desc.UpdatePasswordRequest{
			UserId:          id,
			Password:        password,
			PasswordConfirm: password,
		}

		serviceErr = errors.New("service error")

		res = &emptypb.Empty{}
	)

	tests := []struct {
		name            string
		args            args
		want            *emptypb.Empty
		err             error
		userServiceMock userServiceMockFunc
	}{
		{
			name: "success case",
			args: args{
				req: req,
			},
			want: res,
			err:  nil,
			userServiceMock: func(mc *minimock.Controller) service.UserService {
				mock := mocks.NewUserServiceMock(mc)
				mock.UpdatePasswordMock.Expect(minimock.AnyContext, id, password, password).Return(nil)
				return mock
			},
		},
		{
			name: "service error case",
			args: args{
				req: req,
			},
			want: &emptypb.Empty{},
			err:  serviceErr,
			userServiceMock: func(mc *minimock.Controller) service.UserService {
				mock := mocks.NewUserServiceMock(mc)
				mock.UpdatePasswordMock.Expect(minimock.AnyContext, id, password, password).Return(serviceErr)
				return mock
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			userServiceMock := tt.userServiceMock(mc)
			handler := userHandler.NewHandler(userServiceMock)

			res, err := handler.UpdatePassword(ctx, tt.args.req)
			require.Equal(t, tt.err, err)
			require.Equal(t, tt.want, res)
		})
	}
}
