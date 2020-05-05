FROM node:12-alpine
WORKDIR /app
RUN apk add --no-cache git
RUN chown node:node /app
COPY package.json .
USER node
RUN npm install
COPY webpack* tsconfig.json ./
COPY src src
RUN npm run build

FROM golang:1.14-buster
WORKDIR /app
RUN chown nobody /app
RUN go get -u github.com/gobuffalo/packr/packr
COPY go.mod go.sum ./
RUN go mod download
COPY --from=0 /app/build build
COPY .git .git
COPY res res
COPY server server
COPY main.go .
RUN packr build -ldflags "-X main.gitDescribe=$(git describe --always --tags)" -o peer-calls main.go

FROM debian:buster-slim
WORKDIR /app
COPY --from=1 /app/peer-calls .
USER nobody
EXPOSE 3000
STOPSIGNAL SIGINT
ENTRYPOINT ["./peer-calls"]
