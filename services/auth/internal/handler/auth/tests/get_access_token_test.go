package tests

import (
	"context"
	"errors"
	"testing"

	authHandler "github.com/alesplll/opens3-rebac/services/auth/internal/handler/auth"
	"github.com/alesplll/opens3-rebac/services/auth/internal/service"
	"github.com/alesplll/opens3-rebac/services/auth/pkg/mocks"
	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/auth/v1"
	"github.com/brianvoe/gofakeit"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestGetAccessToken(t *testing.T) {
	type authServiceMockFunc func(mc *minimock.Controller) service.AuthService

	type args struct {
		ctx context.Context
		req *desc.GetAccessTokenRequest
	}

	var (
		ctx = context.Background()
		mc  = minimock.NewController(t)

		refreshToken = gofakeit.UUID()
		accessToken  = gofakeit.UUID()

		req = &desc.GetAccessTokenRequest{
			RefreshToken: refreshToken,
		}

		serviceErr = errors.New("service error")

		res = &desc.GetAccessTokenResponse{
			AccessToken: accessToken,
		}
	)

	tests := []struct {
		name            string
		args            args
		want            *desc.GetAccessTokenResponse
		err             error
		authServiceMock authServiceMockFunc
	}{
		{
			name: "success case",
			args: args{
				ctx: ctx,
				req: req,
			},
			want: res,
			err:  nil,
			authServiceMock: func(mc *minimock.Controller) service.AuthService {
				mock := mocks.NewAuthServiceMock(mc)
				mock.GetAccessTokenMock.Expect(ctx, refreshToken).Return(accessToken, nil)
				return mock
			},
		},
		{
			name: "service error case",
			args: args{
				ctx: ctx,
				req: req,
			},
			want: &desc.GetAccessTokenResponse{},
			err:  serviceErr,
			authServiceMock: func(mc *minimock.Controller) service.AuthService {
				mock := mocks.NewAuthServiceMock(mc)
				mock.GetAccessTokenMock.Expect(ctx, refreshToken).Return("", serviceErr)
				return mock
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			authServiceMock := tt.authServiceMock(mc)
			handler := authHandler.NewHandler(authServiceMock)

			res, err := handler.GetAccessToken(tt.args.ctx, tt.args.req)
			require.Equal(t, tt.err, err)
			require.Equal(t, tt.want, res)
		})
	}
}
