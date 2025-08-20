test:
	go test -coverprofile=coverage $(shell go list ./... | grep -vE '/logs|/mocks|/cmd|/data|/docs')
	go tool cover -func=coverage