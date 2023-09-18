VERSION ?= $(shell git describe --tags)

LD_FLAGS += -X 'github.com/calyptia/cli/cmd/version.Version=${VERSION}'
LD_FLAGS += -w -s
build: get-manifest
	go build -ldflags="${LD_FLAGS}" -o calyptia 

get-manifest: 
	version=$(curl -s https://api.github.com/repos/calyptia/core-operator-releases/releases | jq .[0].tag_name)
	echo ${version}
	gh release download -p manifest.yaml -R calyptia/core-operator ${version}
	mv manifest.yaml cmd/operator