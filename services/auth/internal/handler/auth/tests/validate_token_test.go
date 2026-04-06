package tests

import (
	"context"
	"errors"
	"testing"

	authHandler "github.com/alesplll/opens3-rebac/services/auth/internal/handler/auth"
	"github.com/alesplll/opens3-rebac/services/auth/internal/service"
	"github.com/alesplll/opens3-rebac/services/auth/pkg/mocks"
	"github.com/brianvoe/gofakeit"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestValidateToken(t *testing.T) {
	type authServiceMockFunc func(mc *minimock.Controller) service.AuthService

	var (
		mc = minimock.NewController(t)

		token = gofakeit.UUID()

		ctxWithAuth = metadata.NewIncomingContext(
			context.Background(),
			metadata.Pairs("authorization", "Bearer "+token),
		)
		ctxWithoutMD   = context.Background()
		ctxWithEmptyMD = metadata.NewIncomingContext(
			context.Background(),
			metadata.Pairs(),
		)
	)

	tests := []struct {
		name            string
		ctx             context.Context
		wantErr         bool
		authServiceMock authServiceMockFunc
	}{
		{
			name:    "success case",
			ctx:     ctxWithAuth,
			wantErr: false,
			authServiceMock: func(mc *minimock.Controller) service.AuthService {
				mock := mocks.NewAuthServiceMock(mc)
				mock.ValidateTokenMock.Expect(ctxWithAuth, token).Return(nil)
				return mock
			},
		},
		{
			name:    "service error case",
			ctx:     ctxWithAuth,
			wantErr: true,
			authServiceMock: func(mc *minimock.Controller) service.AuthService {
				mock := mocks.NewAuthServiceMock(mc)
				mock.ValidateTokenMock.Expect(ctxWithAuth, token).Return(errors.New("invalid token"))
				return mock
			},
		},
		{
			name:    "missing metadata",
			ctx:     ctxWithoutMD,
			wantErr: true,
			authServiceMock: func(mc *minimock.Controller) service.AuthService {
				return mocks.NewAuthServiceMock(mc)
			},
		},
		{
			name:    "missing authorization header",
			ctx:     ctxWithEmptyMD,
			wantErr: true,
			authServiceMock: func(mc *minimock.Controller) service.AuthService {
				return mocks.NewAuthServiceMock(mc)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			authServiceMock := tt.authServiceMock(mc)
			handler := authHandler.NewHandler(authServiceMock)

			_, err := handler.ValidateToken(tt.ctx, &emptypb.Empty{})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
