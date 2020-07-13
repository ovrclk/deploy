GITSHA := $(shell git rev-parse HEAD --short)

all: install

mod:
	@go mod tidy

build: mod
	@go build -mod=readonly -o build/deploy main.go

install: mod
	@go build -mod=readonly -o ${GOBIN}/deploy main.go

docker-build:
	@docker build -t jackzampolin/deploy:latest .
	@docker tag jackzampolin/deploy:latest jackzampolin/deploy:${GITSHA}

docker-push:
	@docker push jackzampolin/deploy:latest
	@docker push jackzampolin/deploy:${GITSHA}

docker-run: docker-build
	@docker run -it -v ${HOME}/.akash-deploy:/tmp/config --net=host jackzampolin/deploy:latest

.PHONY: all build install docker-build docker-run