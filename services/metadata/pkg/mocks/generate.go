package mocks

//go:generate ../../bin/minimock -i github.com/alesplll/opens3-rebac/services/metadata/internal/repository.BucketRepository -o . -s "_minimock.go"
//go:generate ../../bin/minimock -i github.com/alesplll/opens3-rebac/services/metadata/internal/repository.ObjectRepository -o . -s "_minimock.go"
//go:generate ../../bin/minimock -i github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db.TxManager -o . -s "_minimock.go"
//go:generate ../../bin/minimock -i github.com/alesplll/opens3-rebac/shared/pkg/go-kit/kafka.Producer -o . -s "_minimock.go"
