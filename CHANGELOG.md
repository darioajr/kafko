# Changelog

All notable changes to kafko will be documented in this file.

## [Unreleased]

### 📚 Documentation

- Kafko tap flag [skip ci]
([8d68f41](https://github.com/darioajr/kafko/commit/8d68f41e9a200f6b95a0c661d845851d3a6072c0))

## [0.4.0] - 2026-05-09

### 📚 Documentation

- Add kafko logo banner to README [skip ci]
([e1bd6fd](https://github.com/darioajr/kafko/commit/e1bd6fd7f9a1890f74b2f4c4a1a3edb08a4de91f))

### 🚀 Features

- Add linux packges and brew tap
([de52a18](https://github.com/darioajr/kafko/commit/de52a18c5d16abe079be789e9312c78d2b941af7))

## [0.3.0] - 2026-05-09

### 🏗️ Build

- Migrate docker config to dockers_v2
Replace the deprecated dockers + docker_manifests pair with a single
    dockers_v2 entry. The new schema:

    - one entry handles all platforms (linux/amd64 + linux/arm64) via buildx
    - pushes to ghcr.io, docker.io and quay.io with both :Version and
      :latest tags
    - generates an SBOM per image by default

    Dockerfile now uses ARG TARGETPLATFORM + COPY ${TARGETPLATFORM}/kafko
    to match the layout goreleaser stages. The Makefile docker target
    mirrors that layout into .docker-context/linux/$arch/kafko so the same
    Dockerfile works for local builds (with podman or docker).

    Net config size went from ~85 lines to ~25.
([bf76706](https://github.com/darioajr/kafko/commit/bf7670642dbb91fb3c531eb0b3dd224af4662f57))

### 🔧 CI/CD

- Bump actions to Node.js 24-compatible majors
GitHub deprecated Node.js 20 actions in Sep 2025; they'll be forced
    to Node 24 from Jun 2026 and removed Sep 2026. Bumping every action
    in the lint, test, build and release workflows to the major that
    ships on Node.js 24:

    - actions/checkout@v6
    - actions/setup-go@v6
    - docker/login-action@v4
    - docker/setup-buildx-action@v4
    - docker/setup-qemu-action@v4
    - goreleaser/goreleaser-action@v7
    - codecov/codecov-action@v6
([2ffe45d](https://github.com/darioajr/kafko/commit/2ffe45d47538196ada27a68911c611e875c97557))

## [0.2.0] - 2026-05-09

### 🐛 Bug Fixes

- File size
([895ca94](https://github.com/darioajr/kafko/commit/895ca94c58ecd7b5815d3d387a89c9ccbed0ee47))

### 🔧 CI/CD

- Bump golangci-lint to v2.12.2 and unpin Go to stable
golangci-lint v2.12.2 ships compiled with Go 1.26.2 and accepts the
    1.26 stdlib without panicking. Drop the Go 1.25.x pin and revert the
    x/crypto v0.43.0 / x/text v0.30.0 downgrades (and the franz-go
    cascade) to latest.

    Net effect on container CVE scans:
    - x/crypto v0.51.0 closes GO-2025-4134 (ssh server unbounded mem)
      and GO-2025-4135 (ssh agent constraint DoS), unreachable in kafko
      but flagged by binary scanners.
    - Go 1.26.3+ (via "stable") closes the 8 stdlib CVEs flagged on
      the kafko:latest image. govulncheck source mode confirms none are
      reachable by kafko code on Linux.
([39ee253](https://github.com/darioajr/kafko/commit/39ee253900611e3fd65cce81bcf6763c1d49362d))

## [0.1.0] - 2026-05-09

### 🐛 Bug Fixes

- **deps:** Pin golang.org/x/crypto to v0.45.0 to satisfy golangci-lint
x/crypto v0.49+ adds chacha20poly1305/fips140only_go1.26.go, gated by
    //go:build go1.26. golangci-lint v2.5.0 ships a go/types built with
    Go 1.25 and panics while parsing the file even though the build tag
    should exclude it.

    Pin keeps franz-go on v1.20.6 / kadm v1.17.2 (transitive cascade).
    Revert when a golangci-lint release built with Go 1.26 lands.
([01aab0a](https://github.com/darioajr/kafko/commit/01aab0a5dc4e6215c8d331bafbd61a8f31483263))
- **docker:** Switch to goreleaser-style binary-copy Dockerfile
Goreleaser builds the binary externally and stages only the executable
    + Dockerfile in the build context. The previous multi-stage Dockerfile
    expected go.mod/go.sum/source in context, which fails with:

      failed to compute cache key: "/go.sum": not found

    Replace with FROM scratch + COPY kafko, sourcing CA certs from a tiny
    alpine builder stage. For `make docker`, add a docker-bin target that
    cross-compiles a linux/$(go env GOARCH) binary into ./bin first.

    Image is now ~5 MB (was ~15 MB) since alpine layer is discarded.
([f4bb75c](https://github.com/darioajr/kafko/commit/f4bb75c948822fec91ab17343cd0ab151190ab00))
- CI
([78dd8b0](https://github.com/darioajr/kafko/commit/78dd8b03e7b2abfcddf1dff7987b3310d85a5b78))
- Go version
([4f7413d](https://github.com/darioajr/kafko/commit/4f7413dbedfb2f1751151afbafacd00acef70ffb))

### 📝 Other

- Initial commit
([488f524](https://github.com/darioajr/kafko/commit/488f5245df17ce0f981371d4e54c92af298488e5))
- Running [/home/runner/golangci-lint-2.5.0-linux-amd64/golangci-lint config path] in [/home/runner/work/kafko/kafko] ...
  Running [/home/runner/golangci-lint-2.5.0-linux-amd64/golangci-lint config verify] in [/home/runner/work/kafko/kafko] ...
  Running [/home/runner/golangci-lint-2.5.0-linux-amd64/golangci-lint run  --timeout=5m] in [/home/runner/work/kafko/kafko] ...
  panic: file requires newer Go version go1.26 (application built with go1.25) [recovered, repanicked]

  goroutine 1914 [running]:
  go/types.(*Checker).handleBailout(0xc00a81e3c0, 0xc002ca7a18)
  	go/types/check.go:467 +0x97
  panic({0x1415540?, 0xc015a89780?})
  	runtime/panic.go:783 +0x132
  github.com/golangci/golangci-lint/v2/pkg/goanalysis.(*loadingPackage).convertError(0xc00198efc0, {0x1bb76c0?, 0xc009217ec0})
  	github.com/golangci/golangci-lint/v2/pkg/goanalysis/runner_loadingpackage.go:484 +0x714
  github.com/golangci/golangci-lint/v2/pkg/goanalysis.(*loadingPackage).loadFromSource.func2({0x1bb76c0?, 0xc009217ec0?})
  	github.com/golangci/golangci-lint/v2/pkg/goanalysis/runner_loadingpackage.go:208 +0x37
  go/types.(*Checker).handleError(0xc00a81e3c0, 0x0, {0x1bb85c0, 0xc003cec360}, 0x97, {0xc0147a47d0, 0x45}, 0x0)
  	go/types/errors.go:221 +0x44a
  go/types.(*error_).report(0xc002ca77b8)
  	go/types/errors.go:148 +0x2cc
  go/types.(*Checker).errorf(0xc0003daa10?, {0x1bb85c0, 0xc003cec360}, 0x6?, {0x1786dc0?, 0xc002ca7850?}, {0xc002ca78c0?, 0x36?, 0x37?})
  	go/types/errors.go:243 +0x150
  go/types.(*Checker).initFiles(0xc00a81e3c0, {0xc002eec3f0, 0x5, 0xc002ca7978?})
  	go/types/check.go:415 +0x771
  go/types.(*Checker).checkFiles(0xc00a81e3c0, {0xc002eec3f0?, 0x6f7145?, 0x1572b20?})
  	go/types/check.go:516 +0x18d
  go/types.(*Checker).Files(0x14c6f20?, {0xc002eec3f0?, 0xc00c66d800?, 0x5?})
  	go/types/check.go:485 +0x75
  github.com/golangci/golangci-lint/v2/pkg/goanalysis.(*loadingPackage).loadFromSource(0xc00198efc0, 0x2)
  	github.com/golangci/golangci-lint/v2/pkg/goanalysis/runner_loadingpackage.go:214 +0x73a
  github.com/golangci/golangci-lint/v2/pkg/goanalysis.(*loadingPackage).loadImportedPackageWithFacts(0xc00198efc0, 0x2)
  	github.com/golangci/golangci-lint/v2/pkg/goanalysis/runner_loadingpackage.go:378 +0x396
  github.com/golangci/golangci-lint/v2/pkg/goanalysis.(*loadingPackage).loadWithFacts(0xc002c9b510?, 0x45dcc9?)
  	github.com/golangci/golangci-lint/v2/pkg/goanalysis/runner_loadingpackage.go:321 +0x128
  github.com/golangci/golangci-lint/v2/pkg/goanalysis.(*loadingPackage).analyze(0xc00198efc0, {0x1bc7208, 0xc002651680}, 0xc0026e50c0, 0x2, 0xc0026b4770)
  	github.com/golangci/golangci-lint/v2/pkg/goanalysis/runner_loadingpackage.go:80 +0x115
  github.com/golangci/golangci-lint/v2/pkg/goanalysis.(*loadingPackage).analyzeRecursive.func1()
  	github.com/golangci/golangci-lint/v2/pkg/goanalysis/runner_loadingpackage.go:61 +0x20e
  sync.(*Once).doSlow(0x0?, 0x0?)
  	sync/once.go:78 +0xac
  sync.(*Once).Do(...)
  	sync/once.go:69
  github.com/golangci/golangci-lint/v2/pkg/goanalysis.(*loadingPackage).analyzeRecursive(0x0?, {0x1bc7208?, 0xc002651680?}, 0x0?, 0x0?, 0x0?)
  	github.com/golangci/golangci-lint/v2/pkg/goanalysis/runner_loadingpackage.go:45 +0x59
  github.com/golangci/golangci-lint/v2/pkg/goanalysis.(*loadingPackage).analyzeRecursive.func1.1(0x0?)
  	github.com/golangci/golangci-lint/v2/pkg/goanalysis/runner_loadingpackage.go:53 +0x30
  created by github.com/golangci/golangci-lint/v2/pkg/goanalysis.(*loadingPackage).analyzeRecursive.func1 in goroutine 566
  	github.com/golangci/golangci-lint/v2/pkg/goanalysis/runner_loadingpackage.go:52 +0xd4

  Error: golangci-lint exit with code 2
([85be057](https://github.com/darioajr/kafko/commit/85be057d7b399bb5eb8812871bb35e77b3d35230))

### 🔧 CI/CD

- Bump golangci-lint to v2.5.0 and migrate config
The previous v1.64.8 was built with Go 1.24 and refused to lint
    against the project's Go target. v2.5.0 ships with Go 1.26, supports
    the v2 config schema, and lets us silence fmt.Fprint* writes via
    errcheck.exclude-functions instead of littering the source with _ = .
([60951af](https://github.com/darioajr/kafko/commit/60951afbae3ae7ebc5a505415645594a19f4e48a))

### 🚀 Features

- Bootstrap kafko CLI with consume/produce/admin and TUI
Native Kafka CLI built around franz-go — no JVM, no librdkafka,
    single static binary.

    CLI commands
    - consume/produce with raw, json (pretty+chroma), hex, base64,
      msgpack and proto (descriptor file) output formats
    - topics list/describe/create/delete and groups list/describe via kadm
    - metadata for cluster + broker overview
    - tui (bubbletea): topic browser with live tail and substring filter

    Config & auth
    - TOML profiles at $XDG_CONFIG_HOME/kafko/config.toml; flags win
    - SASL PLAIN/SCRAM-256/SCRAM-512, TLS and mTLS

    Distribution
    - Multi-stage Dockerfile (FROM scratch, non-root, ~15 MB)
    - goreleaser publishing multi-arch images to ghcr.io, docker.io
      and quay.io plus archives for linux/darwin/windows × amd64/arm64
    - GitHub Actions: ci (lint/vet/test) and tag-driven release
    - Makefile auto-detects docker/podman, snapshot-full uses a podman
      shim so goreleaser runs unchanged
([f3c8105](https://github.com/darioajr/kafko/commit/f3c8105f801c894f12f6e2566d552153264cb74a))

---
*Generated by [git-cliff](https://git-cliff.org)*
