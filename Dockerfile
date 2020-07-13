# Build image
FROM golang:alpine as build-env

# Copy files into $GOPATH for build
RUN mkdir -p /go/src/github.com/jackzampolin/deploy
WORKDIR /go/src/github.com/jackzampolin/deploy

# Install Deps
RUN apk add --update git && \
    go get -u github.com/rakyll/statik

# Copy in application files
COPY . .

# Package Static Assets
RUN statik -src static/

# Build binary
RUN go build -mod=readonly -o build/deploy cmd/*.go

# Production image
FROM alpine:latest

# Copy in files and binary
COPY --from=build-env  /go/src/github.com/jackzampolin/deploy/build/deploy /usr/bin/deploy

# Run the server
CMD ["deploy", "--home", "/tmp/config", "watch"]
