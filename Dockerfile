# Build image
FROM golang:alpine as build-env

# Copy files into $GOPATH for build
RUN mkdir -p /go/src/github.com/ovrclk/deploy
WORKDIR /go/src/github.com/ovrclk/deploy

# Install Deps
RUN apk add --update git && \
    go get -u github.com/rakyll/statik

# Copy in application files
COPY . .

# Build binary
RUN go build -mod=readonly -o build/deploy cmd/*.go

# Production image
FROM alpine:latest

# Copy in files and binary
COPY --from=build-env  /go/src/github.com/ovrclk/deploy/build/deploy /usr/bin/deploy

# Run the server
CMD ["deploy", "--home", "/tmp/config", "watch"]
