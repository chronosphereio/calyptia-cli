
# Calyptia Cloud CLI

## Build instructions

---
```markdown
go env -w GOPRIVATE=github.com/calyptia/cloud
go mod download
go build ./cmd/calyptia
```
---

## Install

Alternatively, you can install only the binary by running:

---
```markdown
go env -w GOPRIVATE=github.com/calyptia/cloud
go install github.com/calyptia/cloud-cli@latest
```
---


## Run

The first command you would want to run is `config set_token` otherwise you will have to always pass `--token` around.<br>
Get a token (API key) from [cloud.calyptia.com](https://cloud.calyptia.com).

---
```markdown
calyptia config set_token TOKEN
```
---

Alternatively, you can set the CALYPTIA_CLOUD_TOKEN environment variable or pass the token on each command,
as an example:
---
```markdown
calyptia get members --token TOKEN
```
---


## Commands

  - `completion`: generate the autocompletion script for the specified shell
    - `bash`: generate the autocompletion script for bash
    - `fish`: generate the autocompletion script for fish
    - `powershell`: generate the autocompletion script for powershell
    - `zsh`: generate the autocompletion script for zsh
  - `config`: Configure Calyptia CLI
    - `set_token TOKEN`: Set the default token so you don't have to specify it on all commands
    - `current_token`: Get the current configured default token
    - `unset_token`: Unset the current configured default token
  - `get`: Display one or many resources
    - `members <options>`: Display latest members from a project
    - `agents <options>`: Display latest agents from a project
    - `aggregators <options>`: Display latest aggregators from a project
    - `pipelines --aggregator AGGREGATOR <options>`: Display latest pipelines from an aggregator
    - `pipeline PIPELINE <options>`: Display a single pipeline
    - `endpoints --pipeline PIPELINE <options>`: Display latest endpoints from a pipeline
    - `pipeline_config_history --pipeline PIPELINE <options>`: Display latest config history from a pipeline
    - `pipeline_status_history --pipeline PIPELINE <options>`: Display latest status history from a pipeline
    - `pipeline_secrets --pipeline PIPELINE <options>`: Display latest pipeline secrets
    - `pipeline_files --pipeline PIPELINE <options>`: Display latest pipeline files
    - `pipeline_file --pipeline PIPELINE --name FILENAME`: Display a single pipeline file
  - `create`: Create pipelines, etc.
    - `pipeline --aggregator AGGREGATOR <options>`: Create a new pipeline
    - `pipeline_file --pipeline PIPELINE --file FILEPATH <options>`: Create a new file within a pipeline
  - `update`: Update aggregators, pipelines, etc
    - `project <options>`: Update the current project
    - `agent AGENT <options>`: Update a single agent by ID or name
    - `aggregator AGGREGATOR <options>`: Update a single aggregator by ID or name
    - `pipeline PIPELINE <options>`: Update a single pipeline by ID or name
    - `pipeline_secret ID VALUE <options>`: Update a pipeline secret value by its ID
    - `pipeline_file --pipeline PIPELINE --file FILENAME <options>`: Update a single file from a pipeline by its name
  - `rollout`: Rollout resources to previous versions
    - `pipeline PIPELINE <options>`: Rollout a pipeline to a previous config
  - `delete`: Delete aggregators, pipelines, etc.
    - `agent AGENT <options>`: Delete a single agent by ID or name
    - `agents  <options>`: Delete many agents from a project
    - `aggregator AGGREGATOR <options>`: Delete a single aggregator by ID or name
    - `pipeline PIPELINE <options>`: Delete a single pipeline by ID or name
  - `top`: Display metrics
    - `project <options>`: Display metrics from the current project
    - `agent AGENT <options>`: Display metrics from an agent
    - `pipeline PIPELINE <options>`: Display metrics from a pipeline
  - `help`: Help about any command
