GOOS := linux
GOARCH := amd64
GOLDFLAGS := -ldflags

# Get a list of all binaries to be built
CMDS := $(shell find ./cmd/ -maxdepth 1 -type d -exec basename {} \; | grep -v cmd)

build: $(CMDS)

$(CMDS):
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-o $@_$(GOOS)_$(GOARCH) \
		./cmd/$@
