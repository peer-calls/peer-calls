FROM node:12-alpine as frontend

# Add dependency instructions and fetch node_modules
COPY package.json package-lock.json /src/
WORKDIR /src

RUN set -ex \
 && apk add --no-cache \
      git \
 && npm ci

# Add the application itself
COPY ./ /src/

RUN set -ex \
 && npm run build


FROM golang:alpine as server

ENV CGO_ENABLED=0

RUN set -ex \
 && apk add --no-cache \
      git \
 && GOPATH=/usr/local go get -u github.com/gobuffalo/packr/packr

# Add dependencies into mod cache
COPY go.mod go.sum /src/
WORKDIR /src

RUN set -ex \
 && go mod download

# Add the application itself and build it
COPY                  ./          /src/
COPY --from=frontend  /src/build/ /src/build/

RUN set -ex \
 && packr build \
      -ldflags "-X main.gitDescribe=$(git describe --always --tags --dirty)" \
      -mod=readonly \
      -o peer-calls


FROM scratch

COPY --from=server /src/peer-calls /usr/local/bin/

EXPOSE 3000/tcp
STOPSIGNAL SIGINT

ENTRYPOINT ["/usr/local/bin/peer-calls"]
