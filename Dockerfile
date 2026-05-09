# Goreleaser dockers_v2 stages binaries under linux/<arch>/kafko in the
# build context, and buildx sets $TARGETPLATFORM during the per-platform
# pass. For local `make docker`, the Makefile mirrors this layout so the
# same Dockerfile works in both flows.

FROM alpine:3.22 AS certs
RUN apk add --no-cache ca-certificates

FROM scratch
ARG TARGETPLATFORM
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY ${TARGETPLATFORM}/kafko /kafko
USER 65532:65532
ENTRYPOINT ["/kafko"]
