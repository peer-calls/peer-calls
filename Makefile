BUILD_FLAGS := -ldflags "-X main.GitDescribe=$(shell git describe --always --tags --dirty)" -o peer-calls

.PHONY: coverage report build pack pack-linux

build:
	go build $(BUILD_FLAGS)

coverage:
	go test ./... -coverprofile=coverage.out

report:
	go tool cover -html=coverage.out

pack:
	packr build $(BUILD_FLAGS)

pack-linux:
	GOOS=linux GOARCH=amd64 packr build $(BUILD_FLAGS)
