
# Calyptia Cloud CLI

[![CI](https://github.com/calyptia/cloud-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/calyptia/cloud-cli/actions/workflows/ci.yml)
[![Codecov](https://codecov.io/gh/calyptia/cli/branch/main/graph/badge.svg?token=TY36W7B87A)](https://codecov.io/gh/calyptia/cli)

This CLI interacts with the [Calyptia Cloud](https://cloud.calyptia.com) service
using the [API Client](https://github.com/calyptia/api).
Futher documentation on how to use the CLI
can be found on [Calyptia Docs](https://docs.calyptia.com).

## Build instructions

---

```markdown
go mod download
go build ./cmd/calyptia
```

---

## Install

You can get the latest release artifacts for the major operating systems
at the [Releases](https://github.com/calyptia/cloud-cli/releases) page.

Alternatively, you can install only the binary by running:

---

```markdown
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
      --token string       Calyptia Cloud Project token (default "eyJUb2tlbklEIjoiOWI5ODIwNzUtYzY4MC00MzdiLWE4YjMtYjU5NjNkMzE4OTUyIiwiUHJvamVjdElEIjoiMDkwZDFhYTEtZGU5Ni00NDZjLTk1NDQtMGUwMGNiNmRkMzkzIn0.JPD_g6oDQdmO_sPlshUJdNefpHT7AMDUjSRjg0x0E61U8-Frh2_ZOCTP93O5UBC9")
  -v, --version            version for calyptia

Use "calyptia [command] --help" for more information about a command.
```
