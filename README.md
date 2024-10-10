# Calyptia Cloud CLI

This CLI interacts with the [Calyptia Cloud](https://core.calyptia.com) service
using the [API Client](https://github.com/chronosphereio/calyptia-api).
Further documentation on how to use the CLI
can be found on [Calyptia Docs](https://docs.chronosphere.io/pipelines).

## Install

You can get the latest release artifacts for the major operating systems
at the [Releases](https://github.com/chronosphereio/calyptia-cli/releases) page.

Alternatively, you can use `Brew`:

```bash
brew tap calyptia/tap
brew install calyptia
```

## Run

The first command you would want to run is `config set_token` otherwise
you will have to always pass `--token` around.

Get a token (API key) from [core.calyptia.com](https://core.calyptia.com).

---

```bash
calyptia config set_token TOKEN
```

---

Alternatively, you can set the CALYPTIA_CLOUD_TOKEN environment variable or
pass the token on each command, as an example:

---

```bash
calyptia get members --token TOKEN
```

---

## Environment variables

A list of the supported environment variables that will override the provided flags.

- CALYPTIA_CLOUD_URL: URL of the cloud API (default: <https://cloud-api.calyptia.com/>)
- CALYPTIA_CLOUD_TOKEN: Cloud project token (default: None)
- CALYPTIA_STORAGE_DIR: Path to store the local configuration (fallback to $HOME/.calyptia)

## Commands

```bash
Calyptia Cloud CLI

Usage:
  calyptia [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  config      Configure Calyptia CLI
  create      Create core instances, pipelines, etc.
  delete      Delete core instances, pipelines, etc.
  get         Display one or many resources
  help        Help about any command
  install     Install calyptia components
  rollout     Rollout resources to previous versions
  uninstall   Uninstall calyptia components
  update      Update core instances, pipelines, etc.
  version     Returns currenty Calyptia CLI version.
  watch       watch for events or logs

Flags:
      --cloud-url string   Calyptia Cloud URL (default "https://cloud-api.calyptia.com")
  -h, --help               help for calyptia
      --token string       Calyptia Cloud Project token (default "check with the 'calyptia config current_token' command")

Use "calyptia [command] --help" for more information about a command.
```
