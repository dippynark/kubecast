GOOS := linux
GOARCH := amd64
GOLDFLAGS := -ldflags

# Get a list of all binaries to be built
CMDS := $(shell find ./cmd/ -maxdepth 1 -type d -exec basename {} \; | grep -v cmd)

DEBUG=1
UID=$(shell id -u)
GID=$(shell id -g)
PWD=$(shell pwd)

DOCKER_FILE ?= Dockerfile
REGISTRY ?= dippynark
APP_NAME ?= kubecast
TAG ?= latest

# If you can use docker without being root, you can do "make SUDO="
SUDO=$(shell docker info >/dev/null 2>&1 || echo "sudo -E")

# Alias targets
all: build
build: bpf $(CMDS) docker_build
bpf: docker_build_image docker_build_bpf install_bpf
docker_build: docker_build_client docker_build_server
docker_push: docker_push_client docker_push_server

# Targets
$(CMDS):
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-o $@_$(GOOS)_$(GOARCH) \
		./cmd/$@

docker_build_image:
	$(SUDO) docker build -t $(REGISTRY)/bpf-builder -f $(DOCKER_FILE) .

docker_build_bpf:
	$(SUDO) docker run --rm -e DEBUG=$(DEBUG) \
		-e CIRCLE_BUILD_URL=$(CIRCLE_BUILD_URL) \
		-v $(PWD):/src:ro \
		-v $(PWD)/bpf:/dist/ \
		-v /usr/src:/usr/src \
		--workdir=/src/bpf \
		$(REGISTRY)/bpf-builder \
		make all
	sudo chown -R $(UID):$(GID) bpf

install_bpf:
	cp bpf/bpf_tty.go pkg/kubecast/bpf_tty.go

docker_clean:
	$(SUDO) docker rmi -f $(REGISTRY)/bpf-builder

docker_build_%:
	cp -f $*_linux_amd64 deploy/docker/$*/$*_linux_amd64
	docker build \
		-f deploy/docker/$*/Dockerfile \
		-t $(REGISTRY)/$(APP_NAME)-$*:$(TAG) \
		deploy/docker/$*

docker_push_%:
	docker push $(REGISTRY)/$(APP_NAME)-$*:$(TAG)
