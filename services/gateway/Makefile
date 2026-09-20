.PHONY: swagger test

swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go -d cmd/server,internal/handler/httpapi -o docs --outputTypes go,json,yaml

test:
	go test ./...
