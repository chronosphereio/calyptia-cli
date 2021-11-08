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
go install github.com/calyptia/cloud-cli@latest
```


## Run

For the time being, the binary needs either `AUTH0_CLIENT_ID` or `--auth0-client-id` to run.<br>
Configure an [auth0](https://auth0.com) project that allows "device code" grant type.

```
calyptia --auth0-client-id ${AUTH0_CLIENT_ID}
```

The first command you would want to run is `login`.

```
calyptia login
```
