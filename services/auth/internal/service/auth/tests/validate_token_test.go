package tests

import (
	"context"
	"errors"
	"testing"

	authService "github.com/alesplll/opens3-rebac/services/auth/internal/service/auth"
	"github.com/alesplll/opens3-rebac/services/auth/pkg/mocks"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/tokens"
	"github.com/brianvoe/gofakeit"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestValidateToken(t *testing.T) {
	ctx := context.Background()
	token := gofakeit.UUID()

	tests := []struct {
		name           string
		token          string
		wantErr        bool
		buildTokenMock func(t *testing.T, mc *minimock.Controller) *mocks.TokenServiceMock
	}{
		{
			name:  "success case",
			token: token,
			buildTokenMock: func(t *testing.T, mc *minimock.Controller) *mocks.TokenServiceMock {
				mock := mocks.NewTokenServiceMock(mc)
				mock.VerifyAccessTokenMock.Expect(ctx, token).Return(&tokens.UserClaims{}, nil)
				return mock
			},
		},
		{
			name:    "invalid token",
			token:   "invalid-token",
			wantErr: true,
			buildTokenMock: func(t *testing.T, mc *minimock.Controller) *mocks.TokenServiceMock {
				mock := mocks.NewTokenServiceMock(mc)
				mock.VerifyAccessTokenMock.Expect(ctx, "invalid-token").Return(nil, errors.New("invalid"))
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

			err := svc.ValidateToken(ctx, tt.token)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
