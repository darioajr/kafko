# Goreleaser builds the binary (with its own toolchain + ldflags) and stages
# it next to this Dockerfile, so we just package it. For local builds, run
# `make docker` — the Makefile compiles ./bin/kafko and copies it to the
# context root before invoking the container engine.

FROM alpine:3.22 AS certs
RUN apk add --no-cache ca-certificates

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY kafko /kafko
USER 65532:65532
ENTRYPOINT ["/kafko"]
