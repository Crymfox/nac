.PHONY: build test clean lint

BINARY_NAME=nac

build:
	go build -o $(BINARY_NAME) ./cmd/nac

test:
	go test ./... -v

clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -f nac.yaml docker-compose.yaml .env.local.example .env.remote.example .gitignore
	rm -rf n8n_workflows n8n_credentials backups .github

lint:
	golangci-lint run

run: build
	./$(BINARY_NAME)
