# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS build
WORKDIR /src

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath \
      -ldflags="-s -w \
        -X github.com/darioajr/kafko/internal/cli.Version=${VERSION} \
        -X github.com/darioajr/kafko/internal/cli.Commit=${COMMIT} \
        -X github.com/darioajr/kafko/internal/cli.Date=${DATE}" \
      -o /out/kafko ./cmd/kafko

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/kafko /kafko
USER 65532:65532
ENTRYPOINT ["/kafko"]
