# Calyptia Cloud CLI

## Build instructions

```
go env -w GOPRIVATE=github.com/calyptia/cloud
go mod download
go build ./cmd/calyptia
```

## Install

Alternatively, you can install only the binary by running:

```
go env -w GOPRIVATE=github.com/calyptia/cloud
go install github.com/calyptia/cloud-cli@latest
```


## Run

For the time being, the binary needs either `CALYPTIA_AUTH0_CLIENT_ID` or `--auth0-client-id` to run.<br>
Configure an [auth0](https://auth0.com) project (native) that allows "device code" grant type.

```
calyptia --auth0-client-id ${CALYPTIA_AUTH0_CLIENT_ID}
```

---
**Better build**

If you don't want to pass `--auth0-client-id` each time, you can inject the value at build time like this:

```
go install -ldflags="-X 'main.defaultAuth0ClientID=YOURS_HERE' github.com/calyptia/cloud-cli@latest
```
---

The first command you would want to run is `login`.

```
calyptia login
```

## Commands

  - `login`: Login by authorizing this CLI with Calyptia Cloud through a browser
  - `completion`: generate the autocompletion script for the specified shell
    - `bash`: generate the autocompletion script for bash
    - `fish`: generate the autocompletion script for fish
    - `powershell`: generate the autocompletion script for powershell
    - `zsh`: generate the autocompletion script for zsh
  - `config`: Configure Calyptia CLI
    - `set_project PROJECT`: Set the default project so you don't have to specify it on all commands
    - `current_project`: Get the current configured default project
    - `unset_project`: Unset the current configured default project
  - `get`: Display one or many resources
    - `projects <options>`: Display latest projects
    - `agents <options>`: Display latest agents from a project
    - `aggregators <options>`: Display latest aggregators from a project
    - `pipelines --aggregator AGGREGATOR <options>`: Display latest pipelines from an aggregator
    - `pipeline PIPELINE <options>`: Display a single pipeline.
    - `endpoints --pipeline PIPELINE <options>`: Display latest endpoints from a pipeline
    - `pipeline_config_history --pipeline PIPELINE <options>`: Display latest config history from a pipeline
    - `pipeline_status_history --pipeline PIPELINE <options>`: Display latest status history from a pipeline
    - `pipeline_secrets --pipeline PIPELINE <options>`: Display latest pipeline secrets.
  - `create`: Create pipelines, etc.
    - `pipeline --aggregator AGGREGATOR <options>`: Create a new pipeline
  - `update`: Update aggregators, pipelines, etc.
    - `project [PROJECT] <options>`: Update a single project by ID or name
    - `agent AGENT <options>`: Update a single agent by ID or name
    - `aggregator AGGREGATOR <options>`: Update a single aggregator by ID or name
    - `pipeline PIPELINE <options>`: Update a single pipeline by ID or name
    - `pipeline_secret ID VALUE <options>`: Update a pipeline secret value by its ID.
  - `rollout`: Rollout resources to previous versions
    - `pipeline PIPELINE <options>`: Rollout a pipeline to a previous config
  - `delete`: Delete aggregators, pipelines, etc.
    - `agent AGENT <options>`: Delete a single agent by ID or name
    - `agents  <options>`: Delete many agents from a project
    - `aggregator AGGREGATOR <options>`: Delete a single aggregator by ID or name
    - `pipeline PIPELINE <options>`: Delete a single pipeline by ID or name
  - `top`: Display metrics
    - `project [PROJECT] <options>`: Display metrics from a project
    - `agent AGENT <options>`: Display metrics from an agent
    - `pipeline PIPELINE <options>`: Display metrics from a pipeline
  - `help`: Help about any command
