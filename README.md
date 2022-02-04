
# Calyptia Cloud CLI

[![CI](https://github.com/calyptia/cloud-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/calyptia/cloud-cli/actions/workflows/ci.yml)
[![Codecov](https://codecov.io/gh/calyptia/cloud-cli/branch/main/graph/badge.svg?token=TY36W7B87A)](https://codecov.io/gh/calyptia/cloud-cli)

## Build instructions

---

```markdown
go env -w GOPRIVATE=github.com/calyptia/api
go mod download
go build ./cmd/calyptia
```

---

## Install

Alternatively, you can install only the binary by running:

---

```markdown
go env -w GOPRIVATE=github.com/calyptia/api
go install github.com/calyptia/cloud-cli@latest
```

---

## Run

The first command you would want to run is `config set_token` otherwise
you will have to always pass `--token` around.

Get a token (API key) from [cloud.calyptia.com](https://cloud.calyptia.com).

---

```markdown
calyptia config set_token TOKEN
```

---

Alternatively, you can set the CALYPTIA_CLOUD_TOKEN environment variable or
pass the token on each command, as an example:

---

```markdown
calyptia get members --token TOKEN
```

---

## Commands

```shell
Calyptia Cloud CLI

Usage:
calyptia [command]

Available Commands:
completion  Generate the autocompletion script for the specified shell
config      Configure Calyptia CLI
create      Create aggregators, pipelines, etc.
delete      Delete aggregators, pipelines, etc.
get         Display one or many resources
help        Help about any command
rollout     Rollout resources to previous versions
top         Display metrics
update      Update aggregators, pipelines, etc.

Flags:
--cloud-url string   Calyptia Cloud URL (default "https://cloud-api.calyptia.com")
-h, --help               help for calyptia
--token string       Calyptia Cloud Project token (default "eyJUb2tlbklEIjoiYWUzNWIzYjQtMWQzZS00NDc0LTgxOTItMWZiNWVhYmUxNTcyIiwiUHJvamVjdElEIjoiM2FlZTlhMmMtMDQwNi00NDkxLTgzNmMtMzYxZjk1ZmU2MTMzIn0.zHxzlUAKo8nl6s4_yyN17HjYNWOfBnlenv2niXywzYh98VJofKtHr3pnEizjO6U2")

Use "calyptia [command] --help" for more information about a command.

```
