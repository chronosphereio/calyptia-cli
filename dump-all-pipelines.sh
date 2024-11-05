#!/bin/bash
set -eu

# Specify the new image to use
OUTPUT_DIR=${OUTPUT_DIR:-$PWD}

echo "Dumping all pipeline config to use: $OUTPUT_DIR"
echo "CLI version: $(calyptia version)"
echo "Endpoint: $(calyptia config current_url)"

# Assumption is we have already configured the CLI token
if [[ -z "$(calyptia config current_token)" ]]; then
    echo "ERROR: no token set - run calyptia config set_token xxx"
    exit 1
fi

for instance in $(calyptia get core_instances -o json | jq -cr '.[].id')
do
    echo "Updating for instance: $instance"
    for pipeline in $(calyptia get pipelines --core-instance "$instance" -o json | jq -cr '.[].id')
    do
        mkdir -p "$OUTPUT_DIR/$instance"
        echo "Dumping pipeline: $pipeline"
        calyptia get pipeline "$pipeline" --only-config > "$OUTPUT_DIR/$instance/$pipeline.yaml"
    done
done
