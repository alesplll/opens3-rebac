package tests

import (
	"context"
	mm_time "time"
	"testing"

	"github.com/alesplll/opens3-rebac/services/users/internal/model"
	"github.com/alesplll/opens3-rebac/services/users/internal/repository"
	userService "github.com/alesplll/opens3-rebac/services/users/internal/service/user"
	"github.com/alesplll/opens3-rebac/services/users/pkg/mocks"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys/codes"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys/validate"
	"github.com/brianvoe/gofakeit"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestCreate(t *testing.T) {
	type userRepoMockFunc func(t *testing.T, mc *minimock.Controller) repository.UserRepository

	type args struct {
		ctx             context.Context
		userInfo        model.UserInfo
		password        string
		passwordConfirm string
	}

	var (
		ctx = context.Background()
		mc  = minimock.NewController(t)

		id       = gofakeit.UUID()
		name     = gofakeit.Name()
		email    = gofakeit.Email()
		password = gofakeit.Password(true, true, true, true, true, 8)

		info = model.UserInfo{
			Name:  name,
			Email: email,
		}
		defaultUserRepositoryMockFunc = func(t *testing.T, mc *minimock.Controller) repository.UserRepository {
			mock := mocks.NewUserRepositoryMock(mc)
			return mock
		}
	)

	tests := []struct {
		name         string
		args         args
		want_id      string
		want_code    codes.Code
		err          error
		userRepoMock userRepoMockFunc
	}{
		{
			name: "success case",
			args: args{
				ctx:             ctx,
				userInfo:        info,
				password:        password,
				passwordConfirm: password,
			},
			want_id:   id,
			want_code: codes.OK,
			err:       nil,
			userRepoMock: func(t *testing.T, mc *minimock.Controller) repository.UserRepository {
				mock := mocks.NewUserRepositoryMock(mc)
				mock.CreateMock.Inspect(func(ctx context.Context, user *model.UserInfo, hashedPassword string, createdAt mm_time.Time) {
					require.NotEmpty(t, hashedPassword)
				}).Return(id, nil)
				return mock
			},
		},
		{
			name: "name is empty case",
			args: args{
				ctx: ctx,
				userInfo: model.UserInfo{
					Name:  "",
					Email: email,
				},
				password:        password,
				passwordConfirm: password,
			},
			want_id:      id,
			want_code:    codes.InvalidArgument,
			err:          nil,
			userRepoMock: defaultUserRepositoryMockFunc,
		},
		{
			name: "email is empty case",
			args: args{
				ctx: ctx,
				userInfo: model.UserInfo{
					Name:  name,
					Email: "",
				},
				password:        password,
				passwordConfirm: password,
			},
			want_id:      id,
			want_code:    codes.InvalidArgument,
			err:          nil,
			userRepoMock: defaultUserRepositoryMockFunc,
		},
		{
			name: "invalid mail case",
			args: args{
				ctx: ctx,
				userInfo: model.UserInfo{
					Name:  name,
					Email: "ivalid_mail@ru",
				},
				password:        password,
				passwordConfirm: password,
			},
			want_id:      id,
			want_code:    codes.InvalidArgument,
			err:          nil,
			userRepoMock: defaultUserRepositoryMockFunc,
		},
		{
			name: "password is empty case",
			args: args{
				ctx:             ctx,
				userInfo:        info,
				password:        "",
				passwordConfirm: "",
			},
			want_id:      id,
			want_code:    codes.InvalidArgument,
			err:          nil,
			userRepoMock: defaultUserRepositoryMockFunc,
		},
		{
			name: "password is not queal case",
			args: args{
				ctx:             ctx,
				userInfo:        info,
				password:        "12345",
				passwordConfirm: "54321",
			},
			want_id:      id,
			want_code:    codes.InvalidArgument,
			err:          nil,
			userRepoMock: defaultUserRepositoryMockFunc,
		},
		{
			name: "password is too short case",
			args: args{
				ctx:             ctx,
				userInfo:        info,
				password:        "1234",
				passwordConfirm: "1234",
			},
			want_id:      id,
			want_code:    codes.InvalidArgument,
			err:          nil,
			userRepoMock: defaultUserRepositoryMockFunc,
		},
	}

	for _, tt := range tests {
		tt := tt // To avoid bugs in parralel tests
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			userRepoMock := tt.userRepoMock(t, mc)
			txManagerMock := mocks.NewTxManagerMock(mc)

			service := userService.NewService(userRepoMock, txManagerMock)

			res_id, err := service.Create(ctx, tt.args.userInfo, tt.args.password, tt.args.passwordConfirm)
			if tt.want_code == codes.OK {
				require.NoError(t, err)
				require.Equal(t, res_id, id)
			} else {
				if sys.IsCommonError(err) {
					ce := sys.GetCommonError(err)
					require.Equal(t, tt.want_code, ce.Code())
				} else if validate.IsValidationError(err) {
					require.Equal(t, tt.want_code, codes.InvalidArgument)
				}
			}
		})
	}
}
