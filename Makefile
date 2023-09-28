VERSION ?= $(shell git describe --tags)

LD_FLAGS += -X 'github.com/calyptia/cli/cmd/version.Version=${VERSION}'
LD_FLAGS += -w -s
build: get-manifest
	go build -ldflags="${LD_FLAGS}" -o calyptia 
