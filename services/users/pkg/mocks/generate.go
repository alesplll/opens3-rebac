package mocks

//go:generate ../../bin/minimock -i github.com/alesplll/opens3-rebac/services/users/internal/service.UserService -o . -s "_minimock.go"
//go:generate ../../bin/minimock -i github.com/alesplll/opens3-rebac/services/users/internal/repository.UserRepository -o . -s "_minimock.go"
//go:generate ../../bin/minimock -i github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db.TxManager -o . -s "_minimock.go"
