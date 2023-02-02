BUILD_FLAGS := -ldflags "-X main.GitDescribe=$(shell git describe --always --tags --dirty)" -o peer-calls

.PHONY: coverage report build build-linux lint lint-env-variables build-assets build-docker

build:
	go build $(BUILD_FLAGS)

build-linux:
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS)

build-assets:
	npm run build

build-linux-docker:
	docker run --rm -t -v $(PWD):/app/peer-calls -w /app/peer-calls -e GOOS=linux -e GOARCH=amd64 golang:1.19.5 go build $(BUILD_FLAGS)

coverage:
	go test ./... -coverprofile=coverage.out

report:
	go tool cover -html=coverage.out

lint: lint-env-variables

lint-env-variables:
	scripts/lint-env-variables.sh
