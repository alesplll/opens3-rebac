package mocks

//go:generate ../../bin/minimock -i github.com/alesplll/opens3-rebac/services/auth/internal/service.AuthService -o . -s "_minimock.go"
//go:generate ../../bin/minimock -i github.com/alesplll/opens3-rebac/services/auth/internal/client/grpc.UserClient -o . -s "_minimock.go"
//go:generate ../../bin/minimock -i github.com/alesplll/opens3-rebac/services/auth/internal/repository.AuthRepository -o . -s "_minimock.go"
//go:generate ../../bin/minimock -i github.com/alesplll/opens3-rebac/shared/pkg/go-kit/tokens.TokenService -o . -s "_minimock.go"
