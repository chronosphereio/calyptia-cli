FROM golang:1.22 as build

# Install certificates
# hadolint ignore=DL3008,DL3015
RUN apt-get update && apt-get install --no-install-recommends -y ca-certificates && update-ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /go/src/github.com/chronosphereio/calyptia-cli

# Now do the rest of the source code - this way we can speed up local iteration
COPY . .

RUN CGO_ENABLED=0 go build -a -gcflags=all="-C -l -B" -ldflags="-w -s" -tags netgo,osusergo -o /calyptia

FROM scratch as production

LABEL org.opencontainers.image.title="chronosphereio/calyptia-cli" \
      org.opencontainers.image.description="Calyptia CLI" \
      org.opencontainers.image.maintainer="CI <ci@calyptia.com>" \
      org.opencontainers.image.url="https://calyptia.com"

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /calyptia /calyptia

ENTRYPOINT [ "/calyptia" ]
