GITSHA := `git rev-parse --short HEAD`
GO := GO111MODULE=on go
SCRIPTS := "./scripts"

###########################################
###########      BUILD       ##############
###########################################

all: install

mod:
	@$(GO) mod tidy

mod-download:
	@$(GO) mod download

build: mod
	@$(GO) build -mod=readonly -o build/deploy main.go

install: mod
	@$(GO) build -mod=readonly -o ${GOBIN}/deploy main.go

docker-build:
	@docker build -t ovrclk/deploy:latest .
	@docker tag ovrclk/deploy:latest ovrclk/deploy:${GITSHA}

docker-push:
	@docker push ovrclk/deploy:latest
	@docker push ovrclk/deploy:${GITSHA}

docker-run: docker-build
	@docker run -it -v ${HOME}/.akash-deploy:/tmp/config --net=host ovrclk/deploy:latest

install-deps: mod-download
	$(GO) install github.com/vektra/mockery/.../
	$(GO) install k8s.io/code-generator/...
	$(GO) install sigs.k8s.io/kind
	$(GO) install golang.org/x/tools/cmd/stringer

###########################################
############      DEMO       ##############
###########################################

install-akash:
	@$(SCRIPTS)/akash.sh

install-akash-local:
	@$(SCRIPTS)/akash.sh local

start-chain:
	@$(SCRIPTS)/chain.sh skip

create-kind:
	@$(SCRIPTS)/kind.sh create

stop-kind:
	@$(SCRIPTS)/kind.sh delete

stop-provider:
	@killall -SIGTERM akashctl

stop-chain:
	@killall -SIGTERM akashd

chain-logs:
	@tail -f ./data/chain.log

provider-logs:
	@tail -f ./data/provider.log

stop-all: stop-kind stop-provider stop-chain

create-provider:
	@echo "Creating akash provider..."
	@akashctl --home ./data/client tx provider create --from provider ./scripts/provider.yaml -y &> /dev/null
	@sleep 5

create-deploy:
	@$(SCRIPTS)/setup-deploy.sh skip

run-provider:
	@$(SCRIPTS)/provider.sh 

demo: install install-akash start-chain create-kind create-provider run-provider create-deploy

demo-local: install install-akash-local start-chain create-kind create-provider run-provider create-deploy

demo-reset: start-chain create-kind create-provider run-provider create-deploy

.PHONY: all build install docker-build docker-run