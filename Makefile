default: docker

IMAGE := covenantsql/covenantsql
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

docker: status
	docker build \
		--tag $(IMAGE):$(VERSION) \
		--tag $(IMAGE):latest \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg VERSION=$(VERSION) \
		.

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

bp:
	./build.sh bp

miner:
	./build.sh miner

client:
	./build.sh client

all:
	./build.sh

.PHONY: status docker save start stop logs push bp miner client all
