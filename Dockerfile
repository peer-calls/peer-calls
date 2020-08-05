FROM node:12-alpine as frontend

COPY ./ /src/
wORKDIR /src

RUN set -ex \
 && apk add --no-cache \
      git \
 && npm ci \
 && npm run build


FROM golang:alpine as server

ENV CGO_ENABLED=0

RUN set -ex \
 && apk add --no-cache \
      git \
 && GOPATH=/usr/local go get -u github.com/gobuffalo/packr/packr

COPY                  ./          /src/
COPY --from=frontend  /src/build/ /src/build/
wORKDIR /src

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
