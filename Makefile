VERSION ?= $(shell git describe --tags)

LD_FLAGS += -X 'github.com/calyptia/cli/commands/version.Version=${VERSION}'
LD_FLAGS += -w -s
build:
	go build -ldflags="${LD_FLAGS}" -o calyptia
