package tests

import (
	"context"
	"errors"
	"testing"

	domainerrors "github.com/alesplll/opens3-rebac/services/auth/internal/errors/domain"
	authService "github.com/alesplll/opens3-rebac/services/auth/internal/service/auth"
	"github.com/alesplll/opens3-rebac/services/auth/pkg/mocks"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/tokens"
	"github.com/brianvoe/gofakeit"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestGetAccessToken(t *testing.T) {
	type args struct {
		ctx          context.Context
		refreshToken string
	}

	ctx := context.Background()
	refreshToken := gofakeit.UUID()
	accessToken := gofakeit.UUID()
	userID := gofakeit.UUID()
	email := gofakeit.Email()

	tokenErr := errors.New("token generation error")

	tests := []struct {
		name           string
		args           args
		wantToken      string
		wantErr        error
		buildTokenMock func(t *testing.T, mc *minimock.Controller) *mocks.TokenServiceMock
	}{
		{
			name: "success case",
			args: args{ctx: ctx, refreshToken: refreshToken},
			wantToken: accessToken,
			buildTokenMock: func(t *testing.T, mc *minimock.Controller) *mocks.TokenServiceMock {
				mock := mocks.NewTokenServiceMock(mc)
				mock.VerifyRefreshTokenMock.Expect(ctx, refreshToken).Return(&tokens.UserClaims{
					UserId: userID,
					Email:  email,
				}, nil)
				mock.GenerateAccessTokenMock.Return(accessToken, nil)
				return mock
			},
		},
		{
			name:    "invalid refresh token",
			args:    args{ctx: ctx, refreshToken: "invalid"},
			wantErr: domainerrors.ErrInvalidRefreshToken,
			buildTokenMock: func(t *testing.T, mc *minimock.Controller) *mocks.TokenServiceMock {
				mock := mocks.NewTokenServiceMock(mc)
				mock.VerifyRefreshTokenMock.Expect(ctx, "invalid").Return(nil, errors.New("invalid"))
				return mock
			},
		},
		{
			name:    "token generation error",
			args:    args{ctx: ctx, refreshToken: refreshToken},
			wantErr: tokenErr,
			buildTokenMock: func(t *testing.T, mc *minimock.Controller) *mocks.TokenServiceMock {
				mock := mocks.NewTokenServiceMock(mc)
				mock.VerifyRefreshTokenMock.Expect(ctx, refreshToken).Return(&tokens.UserClaims{
					UserId: userID,
					Email:  email,
				}, nil)
				mock.GenerateAccessTokenMock.Return("", tokenErr)
				return mock
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			tokenMock := tt.buildTokenMock(t, mc)
			userClientMock := mocks.NewUserClientMock(mc)
			repoMock := mocks.NewAuthRepositoryMock(mc)

			svc := authService.NewService(userClientMock, tokenMock, repoMock)

			token, err := svc.GetAccessToken(tt.args.ctx, tt.args.refreshToken)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Empty(t, token)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantToken, token)
		})
	}
}
