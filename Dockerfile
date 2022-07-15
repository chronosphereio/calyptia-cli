FROM golang:1.18 as build

WORKDIR /go/src/github.com/calyptia/cli

# Now do the rest of the source code - this way we can speed up local iteration
COPY . .

RUN go build -ldflags "-w -s" -tags netgo,osusergo -o /calyptia ./cmd/calyptia

FROM scratch as production
LABEL org.opencontainers.image.title="calyptia/cli" \
      org.opencontainers.image.description="Calyptia CLI" \
      org.opencontainers.image.maintainer="Jorge Niedbalski <j@calyptia.com>" \
      org.opencontainers.image.url="https://calyptia.com"

COPY --from=build /calyptia /calyptia

ENTRYPOINT [ "/calyptia" ]