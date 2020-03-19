BUILD_FLAGS := -ldflags "-X main.gitDescribe=$(shell git describe --always --tags --dirty)"

coverage:
	go test ./... -coverprofile=coverage.out

report:
	go tool cover -html=coverage.out

build:
	go build $(BUILD_FLAGS)

pack: css-prod
	packr build $(BUILD_FLAGS)

pack-linux: css-prod
	GOOS=linux GOARCH=amd64 packr build $(BUILD_FLAGS)
