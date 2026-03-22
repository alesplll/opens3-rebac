package model

type UserInfo struct {
	UserId string
	Email  string
}

func (ui UserInfo) GetUserID() string {
	return ui.UserId
}

func (ui UserInfo) GetEmail() string {
	return ui.Email
}
