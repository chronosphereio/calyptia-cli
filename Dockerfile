FROM golang:1.19 as build

# Install certificates
# hadolint ignore=DL3008,DL3015
RUN apt-get update && apt-get install --no-install-recommends -y ca-certificates && update-ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /go/src/github.com/calyptia/cli

# Now do the rest of the source code - this way we can speed up local iteration
COPY . .

RUN go build -ldflags "-w -s" -tags netgo,osusergo -o /calyptia ./cmd/calyptia

FROM scratch as production
LABEL org.opencontainers.image.title="calyptia/cli" \
      org.opencontainers.image.description="Calyptia CLI" \
      org.opencontainers.image.maintainer="Jorge Niedbalski <j@calyptia.com>" \
      org.opencontainers.image.url="https://calyptia.com"

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /calyptia /calyptia

ENTRYPOINT [ "/calyptia" ]
