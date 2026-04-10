package config

type JWTVerifierConfig interface {
	AccessTokenSecretKey() string
	RefreshTokenSecretKey() string
}

type jwtVerifierAdapter struct {
	cfg JWTConfig
}

func NewJWTVerifierConfig(cfg JWTConfig) JWTVerifierConfig {
	return &jwtVerifierAdapter{cfg: cfg}
}

func (a *jwtVerifierAdapter) AccessTokenSecretKey() string {
	return a.cfg.AccessTokenSecretKey()
}

func (a *jwtVerifierAdapter) RefreshTokenSecretKey() string {
	return ""
}
