#!/bin/bash
set -eu

# Specify the new image to use
NEW_IMAGE=${NEW_IMAGE:-$1}

echo "Updating all pipelines to use image: $NEW_IMAGE"
echo "CLI version: $(calyptia version)"
echo "Endpoint: $(calyptia config current_url)"

# Assumption is we have already configured the CLI token
if [[ -z "$(calyptia config current_token)" ]]; then
    echo "ERROR: no token set - run calyptia config set_token xxx"
    exit 1
fi

for instance in $(calyptia get core_instances --last 0 -o json | jq -cr '.[].id')
do
    echo "Updating for instance: $instance"
    for pipeline in $(calyptia get pipelines  --last 0 --core-instance "$instance" -o json | jq -cr '.[].id')
    do
        echo "Updating pipeline: $pipeline"
        calyptia update pipeline "$pipeline" --image "$NEW_IMAGE"
    done
done
