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

push:
	docker push $(IMAGE):$(VERSION)
	docker push $(IMAGE):latest



branch := $(shell git rev-parse --abbrev-ref HEAD)
builddate := $(shell date +%Y%m%d%H%M%S)

unamestr := $(shell uname)

ifeq ($(unamestr),"Linux")
platform := "linux"
endif

version := $(branch)-$(GIT_COMMIT)-$(builddate)

tags := $(platform) sqlite_omit_load_extension
testtags := $(platform) sqlite_omit_load_extension testbinary
test_flags := -coverpkg github.com/CovenantSQL/CovenantSQL/... -cover -race -c

ldflags_role_bp := -X main.version=$(version) -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=B $$GOLDFLAGS
ldflags_role_miner := -X main.version=$(version) -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=M $$GOLDFLAGS
ldflags_role_client := -X main.version=$(version) -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=C $$GOLDFLAGS

GOTEST := CGO_ENABLED=1 go test $(test_flags) -tags "$(testtags)"
GOBUILD := CGO_ENABLED=1 go build -tags "$(tags)"

bp_test:
	$(GOTEST) \
		-ldflags "$(ldflags_role_bp)" \
		-o bin/cqld.test \
		github.com/CovenantSQL/CovenantSQL/cmd/cqld

bp_bin:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_bp)" \
		-o bin/cqld \
		github.com/CovenantSQL/CovenantSQL/cmd/cqld

miner_test:
	$(GOTEST) \
		-ldflags "$(ldflags_role_miner)" \
		-o bin/cql-minerd.test \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-minerd

miner_bin:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_miner)" \
		-o bin/cql-minerd \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-minerd

observer_test:
	$(GOTEST) \
		-ldflags "$(ldflags_role_client)" \
		-o bin/cql-observer.test \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-observer

observer_bin:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client)" \
		-o bin/cql-observer \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-observer

utils_bin:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client)" \
		-o bin/cql-utils \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-utils

cli_bin:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client)" \
		-o bin/cql \
		github.com/CovenantSQL/CovenantSQL/cmd/cql

fuse_bin:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client)" \
		-o bin/cql-fuse \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-fuse

adapter_bin:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client)" \
		-o bin/cql-adapter \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-adapter

mysql_adapter_bin:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client)" \
		-o bin/cql-mysql-adapter \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-mysql-adapter

faucet_bin:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client)" \
		-o bin/cql-faucet \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-faucet

explorer_bin:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client)" \
		-o bin/cql-explorer \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-explorer

bp: bp_bin bp_test

miner: miner_bin miner_test

observer: observer_bin observer_test

client: utils_bin cli_bin fuse_bin adapter_bin mysql_adapter_bin faucet_bin explorer_bin

all: bp miner observer client

.PHONY: status start stop logs push bp_bin bp_test miner_bin miner_test utils_bin cli_bin fuse_bin adapter_bin mysql_adapter_bin faucet_bin explorer_bin
