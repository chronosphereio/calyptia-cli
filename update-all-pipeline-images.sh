#!/bin/bash
set -eu

filter() {
    local command="$1"
    local field="$2"
    local regex="$3"
    $command | jq -cr --arg field "$field" --arg pattern "$regex" '
        map(select(.[$field] | tostring | test($pattern)))
    '
}

print_usage() {
    echo "Usage: ./update-all-pipeline-images.sh <IMAGE>
  Flags:
    -d | -dryrun: Enables dryrun. This prints pipelines to be updated but will not apply changes
  Optional ENVVAR filters:
    FILTER_INSTANCE_BY_NAME: Only considers pipelines in core instances whose name matches the provided regex
    FILTER_INSTANCE_BY_ID: Only considers pipelines in core instances whose id matches the provided regex
    FILTER_PIPE_BY_NAME: Only applies to pipelines whose name matches the provided regex
    FILTER_PIPE_BY_ID: Only applies to pipelines whose id matches the provided regex
  Examples:
    FILTER_INSTANCE_BY_NAME=dev.* FILTER_PIPE_BY_NAME=tcp.* ./update-all-pipeline-images.sh new_image
      This will update all pipelines within dev.* with names matching tcp.* to new_image
    ./update-all-pipeline-images.sh new_image
      This will update all pipelines to new_image"
}

dryrun=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|-help)
      print_usage
      exit 0
      ;;
    -d|-dryrun)
      dryrun=true
      shift
      ;;
    *)
      NEW_IMAGE="${NEW_IMAGE:-${1-}}"
      shift
      ;;
  esac
done

FILTER_INSTANCE_BY_NAME="${FILTER_INSTANCE_BY_NAME:-}"
FILTER_INSTANCE_BY_ID="${FILTER_INSTANCE_BY_ID:-}"
FILTER_PIPE_BY_NAME="${FILTER_PIPE_BY_NAME:-}"
FILTER_PIPE_BY_ID="${FILTER_PIPE_BY_ID:-}"

if [[ -z "${NEW_IMAGE+x}" ]]
then
    echo "No IMAGE set"
    print_usage
    exit 1
fi

if [[ "$dryrun" = true ]]
then
    echo "NOTE: dryrun enabled, no updates will be applied"
fi

echo "Updating all pipelines to use image: $NEW_IMAGE"
echo "CLI version: $(calyptia version)"
echo "Endpoint: $(calyptia config current_url)"

# Assumption is we have already configured the CLI token
if [[ -z "$(calyptia config current_token)" ]]; then
    echo "ERROR: no token set - run calyptia config set_token xxx"
    exit 1
fi

CORE_INSTANCES=$(calyptia get core_instances -o json)

# Filter core instances by name if $FILTER_INSTANCE_BY_NAME is provided.
if [[ -n "$FILTER_INSTANCE_BY_NAME" ]]
then
    CORE_INSTANCES=$(filter "echo $CORE_INSTANCES" name "$FILTER_INSTANCE_BY_NAME")
fi

# Filter core instances by id if $FILTER_INSTANCE_BY_ID is provided.
if [[ -n "$FILTER_INSTANCE_BY_ID" ]]
then
    CORE_INSTANCES=$(filter "echo $CORE_INSTANCES" id "$FILTER_INSTANCE_BY_ID")
fi


for instance in $(echo "$CORE_INSTANCES" | jq -cr '.[]')
do
    instance_name=$(echo "$instance" | jq -cr '.name')
    instance_id=$(echo "$instance" | jq -cr '.id')
    PIPELINES=$(calyptia get pipelines --core-instance "$instance_id" -o json)
    # Filter pipelines by name if $FILTER_PIPE_BY_NAME is provided.
    if [[ -n "$FILTER_PIPE_BY_NAME" ]]
    then
        PIPELINES=$(filter "echo $PIPELINES" name "$FILTER_PIPE_BY_NAME")
    fi

    # Filter pipelines by id if $FILTER_PIPE_BY_ID is provided.
    if [[ -n "$FILTER_PIPE_BY_ID" ]]
    then
        PIPELINES=$(filter "echo $PIPELINES" id "$FILTER_PIPE_BY_ID")
    fi

    if [[ "$PIPELINES" != [] ]]
    then
        echo "Updating pipelines in core instance $instance_name ($instance_id)"
        for pipeline in $(echo "$PIPELINES" | jq -cr '.[] | {name,id}')
        do
            pipe_name=$(echo "$pipeline" | jq -cr '.name')
            pipe_id=$(echo "$pipeline" | jq -cr '.id')

            if [[ "$dryrun" = true ]]
            then
                echo "  Updating pipeline $pipe_name ($pipe_id) (dryrun)"
            else
                echo "  Updating pipeline $pipe_name ($pipe_id)"
                calyptia update pipeline "$pipe_id" --image "$NEW_IMAGE"
            fi
        done
    else
        echo "No pipelines in core instance $instance_name ($instance_id) match filter, skipping"
    fi
done
