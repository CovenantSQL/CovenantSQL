default: all

# Do a parallel build with multiple jobs, based on the number of CPUs online
# in this system: 'make -j8' on a 8-CPU system, etc.
ifeq ($(JOBS),)
  JOBS := $(shell grep -c ^processor /proc/cpuinfo 2>/dev/null)
  ifeq ($(JOBS),)
    JOBS := $(shell sysctl -n hw.logicalcpu 2>/dev/null)
    ifeq ($(JOBS),)
      JOBS := 1
    endif
  endif
endif

use_all_cores:
	make -j$(JOBS) all

BUILDER := covenantsql/covenantsql-builder
IMAGE := covenantsql/covenantsql
OB_IMAGE := covenantsql/covenantsql-observer

GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
GIT_DIRTY ?= $(shell test -n "`git status --porcelain`" && echo "+CHANGES" || true)
GIT_DESCRIBE ?= $(shell git describe --tags --always)

COMMIT := $(GIT_COMMIT)$(GIT_DIRTY)
VERSION := $(GIT_DESCRIBE)
SHIP_VERSION := $(shell docker image inspect -f "{{ .Config.Labels.version }}" $(IMAGE):latest 2>/dev/null)
IMAGE_TAR := $(subst /,_,$(IMAGE)).$(SHIP_VERSION).tar
IMAGE_TAR_GZ := $(IMAGE_TAR).gz

status:
	@echo "Commit: $(COMMIT) Version: $(VERSION) Ship Version: $(SHIP_VERSION)"


builder: status
	docker build \
		--tag $(BUILDER):$(VERSION) \
		--tag $(BUILDER):latest \
		--build-arg BUILD_ARG=use_all_cores \
		-f docker/builder.Dockerfile \
		.

runner: builder
	docker build \
		--tag $(IMAGE):$(VERSION) \
		--tag $(IMAGE):latest \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg VERSION=$(VERSION) \
		-f docker/Dockerfile \
		.

observer_docker: builder
	docker build \
		--tag $(OB_IMAGE):$(VERSION) \
		--tag $(OB_IMAGE):latest \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg VERSION=$(VERSION) \
		-f docker/observer.Dockerfile \
		.

docker: runner observer_docker

docker_clean: status
	docker rmi -f $(BUILDER):latest
	docker rmi -f $(IMAGE):latest
	docker rmi -f $(OB_IMAGE):latest
	docker rmi -f $(BUILDER):$(VERSION)
	docker rmi -f $(IMAGE):$(VERSION)
	docker rmi -f $(OB_IMAGE):$(VERSION)


save: status
ifeq ($(SHIP_VERSION),)
	$(error No version to ship, please build first)
endif
	docker save $(IMAGE):$(SHIP_VERSION) > $(IMAGE_TAR)
	tar zcf $(IMAGE_TAR_GZ) $(IMAGE_TAR)

start:
	docker-compose down
	docker-compose up --no-start
	docker-compose start

stop:
	docker-compose down

logs:
	docker-compose logs -f --tail=10

push_testnet:
	docker tag $(OB_IMAGE):$(VERSION) $(OB_IMAGE):testnet
	docker push $(OB_IMAGE):testnet
	docker tag $(IMAGE):$(VERSION) $(IMAGE):testnet
	docker push $(IMAGE):testnet

push_bench:
	docker tag $(OB_IMAGE):$(VERSION) $(OB_IMAGE):bench
	docker push $(OB_IMAGE):bench
	docker tag $(IMAGE):$(VERSION) $(IMAGE):bench
	docker push $(IMAGE):bench

push_staging:
	docker tag $(OB_IMAGE):$(VERSION) $(OB_IMAGE):staging
	docker push $(OB_IMAGE):staging
	docker tag $(IMAGE):$(VERSION) $(IMAGE):staging
	docker push $(IMAGE):staging


push:
	docker push $(OB_IMAGE):$(VERSION)
	docker push $(OB_IMAGE):latest
	docker push $(IMAGE):$(VERSION)
	docker push $(IMAGE):latest



branch := $(shell git rev-parse --abbrev-ref HEAD)
builddate := $(shell date +%Y%m%d%H%M%S)

unamestr := $(shell uname)

ifeq ($(unamestr),Linux)
platform := linux
endif

version := $(branch)-$(GIT_COMMIT)-$(builddate)

tags := $(platform) sqlite_omit_load_extension
testtags := $(tags) testbinary
test_flags := -coverpkg github.com/CovenantSQL/CovenantSQL/... -cover -race -c

ldflags_role_bp := -X main.version=$(version) -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=B $$GOLDFLAGS
ldflags_role_miner := -X main.version=$(version) -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=M $$GOLDFLAGS
ldflags_role_client := -X main.version=$(version) -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=C $$GOLDFLAGS
ldflags_role_client_simple_log := $(ldflags_role_client) -X github.com/CovenantSQL/CovenantSQL/utils/log.SimpleLog=Y

GOTEST := CGO_ENABLED=1 go test $(test_flags) -tags "$(testtags)"
GOBUILD := CGO_ENABLED=1 go build -tags "$(tags)"

bin/cqld.test:
	$(GOTEST) \
		-ldflags "$(ldflags_role_bp)" \
		-o bin/cqld.test \
		github.com/CovenantSQL/CovenantSQL/cmd/cqld

bin/cqld:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_bp)" \
		-o bin/cqld \
		github.com/CovenantSQL/CovenantSQL/cmd/cqld

bin/cql-minerd.test:
	$(GOTEST) \
		-ldflags "$(ldflags_role_miner)" \
		-o bin/cql-minerd.test \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-minerd

bin/cql-minerd:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_miner)" \
		-o bin/cql-minerd \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-minerd

bin/cql-observer.test:
	$(GOTEST) \
		-ldflags "$(ldflags_role_client)" \
		-o bin/cql-observer.test \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-observer

bin/cql-observer:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client)" \
		-o bin/cql-observer \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-observer

bin/cql-utils:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client_simple_log)" \
		-o bin/cql-utils \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-utils

bin/cql:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client_simple_log)" \
		-o bin/cql \
		github.com/CovenantSQL/CovenantSQL/cmd/cql

bin/cql-fuse:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client_simple_log)" \
		-o bin/cql-fuse \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-fuse

bin/cql-adapter:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client)" \
		-o bin/cql-adapter \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-adapter

bin/cql-mysql-adapter:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client)" \
		-o bin/cql-mysql-adapter \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-mysql-adapter

bin/cql-faucet:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client)" \
		-o bin/cql-faucet \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-faucet

bin/cql-explorer:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client)" \
		-o bin/cql-explorer \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-explorer

bp: bin/cqld.test bin/cqld

miner: bin/cql-minerd.test bin/cql-minerd

observer: bin/cql-observer.test bin/cql-observer

client: bin/cql-utils bin/cql bin/cql-fuse bin/cql-adapter bin/cql-mysql-adapter bin/cql-faucet bin/cql-explorer

all: bp miner observer client

clean:
	rm -rf bin/cql*
	rm -f *.cover.out
	rm -f coverage.txt

.PHONY: status start stop logs push push_testnet clean \
	bin/cqld.test bin/cqld bin/cql-minerd.test bin/cql-minerd bin/cql-utils bin/cql-observer bin/cql-observer.test \
	bin/cql bin/cql-fuse bin/cql-adapter bin/cql-mysql-adapter bin/cql-faucet bin/cql-explorer
