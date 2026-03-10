package tokens

import "github.com/golang-jwt/jwt/v5"

type UserClaims struct {
	jwt.RegisteredClaims
	UserId    string `json:"user_id"`
	Email     string `json:"email"`
	TokenType string `json:"token_type"` // refresh or access
}

type UserInfo interface {
	GetUserID() string
	GetEmail() string
}
