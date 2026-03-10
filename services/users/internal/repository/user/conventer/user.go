package conventer

import (
	"github.com/alesplll/opens3-rebac/services/users/internal/model"
	modelRepo "github.com/alesplll/opens3-rebac/services/users/internal/repository/user/model"
)

func FromRepoToModelUserInfo(userInfo *modelRepo.UserInfo) *model.UserInfo {
	return &model.UserInfo{
		Name:  userInfo.Name,
		Email: userInfo.Email,
	}
}

func FromModelToRepoUserInfo(userInfo *model.UserInfo) *modelRepo.UserInfo {
	return &modelRepo.UserInfo{
		Name:  userInfo.Name,
		Email: userInfo.Email,
	}
}

func FromRepoToModelUser(user *modelRepo.User) *model.User {
	return &model.User{
		Id:        user.Id,
		UserInfo:  *FromRepoToModelUserInfo(&user.UserInfo),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func FromModelToRepoUser(user *model.User) *modelRepo.User {
	return &modelRepo.User{
		Id:        user.Id,
		UserInfo:  *FromModelToRepoUserInfo(&user.UserInfo),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
