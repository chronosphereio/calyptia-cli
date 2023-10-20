#!/bin/bash
set -eu
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

CONFIG=$(curl -sSfl https://raw.githubusercontent.com/calyptia/core-product-release/main/component-config.json)
echo "$CONFIG" | jq .

export NEW_VERSION=${NEW_VERSION:-$(echo "$CONFIG" | jq -r .versions.core_operator)}
sed -i -E "s/DefaultCoreOperatorDockerImageTag = .*$/DefaultCoreOperatorDockerImageTag = \"$NEW_VERSION\"/g" "$SCRIPT_DIR/"cmd/utils/utils.go
sed -i -E "s/DefaultCoreOperatorToCloudDockerImageTag = .*$/DefaultCoreOperatorToCloudDockerImageTag = \"$NEW_VERSION\"/g" "$SCRIPT_DIR/"cmd/utils/utils.go
sed -i -E "s/DefaultCoreOperatorFromCloudDockerImageTag = .*$/DefaultCoreOperatorFromCloudDockerImageTag = \"$NEW_VERSION\"/g" "$SCRIPT_DIR/"cmd/utils/utils.go
