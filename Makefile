GOOS := linux
GOARCH := amd64
GOLDFLAGS := -ldflags

# Get a list of all binaries to be built
CMDS := $(shell find ./cmd/ -maxdepth 1 -type d -exec basename {} \; | grep -v cmd)

DEBUG=1
UID=$(shell id -u)
GID=$(shell id -g)
PWD=$(shell pwd)

DOCKER_FILE?=Dockerfile
DOCKER_IMAGE?=dippynark/bpf-builder

# If you can use docker without being root, you can do "make SUDO="
SUDO=$(shell docker info >/dev/null 2>&1 || echo "sudo -E")

all: build

build: bpf $(CMDS)

$(CMDS):
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-o $@_$(GOOS)_$(GOARCH) \
		./cmd/$@

bpf: build-docker-image build-bpf-object install-generated-go

build-docker-image:
	$(SUDO) docker build -t $(DOCKER_IMAGE) -f $(DOCKER_FILE) .

build-bpf-object:
	$(SUDO) docker run --rm -e DEBUG=$(DEBUG) \
		-e CIRCLE_BUILD_URL=$(CIRCLE_BUILD_URL) \
		-v $(PWD):/src:ro \
		-v $(PWD)/bpf:/dist/ \
		-v $(PWD)/linux-headers:/linux-headers \
		--workdir=/src/bpf \
		$(DOCKER_IMAGE) \
		make all
	sudo chown -R $(UID):$(GID) bpf

install-generated-go:
	cp bpf/bpf_tty.go pkg/kubepf/bpf_tty.go

delete-docker-image:
	$(SUDO) docker rmi -f $(DOCKER_IMAGE)
