# Contributing to kafko

Thanks for your interest! kafko is small enough that contributions are easy to land — here's the playbook.

## Development setup

```sh
git clone https://github.com/darioajr/kafko.git
cd kafko
make tidy
make build
make test
```

Requires Go 1.23+.

## Running against a local Kafka

The fastest path is [Redpanda](https://redpanda.com) in Docker:

```sh
docker run -d --name redpanda -p 9092:9092 \
  redpandadata/redpanda:latest \
  redpanda start --overprovisioned --smp 1 --memory 1G --reserve-memory 0M \
    --node-id 0 --check=false \
    --kafka-addr PLAINTEXT://0.0.0.0:9092 \
    --advertise-kafka-addr PLAINTEXT://localhost:9092

./bin/kafko -b localhost:9092 topics create demo
echo "hello" | ./bin/kafko -b localhost:9092 produce -t demo
./bin/kafko -b localhost:9092 consume -t demo --from-beginning -n 1
```

## Coding standards

* `make lint` and `make test` must pass.
* Format with `gofmt` (CI enforces it). `make fmt` runs it for you.
* Keep `internal/...` truly internal — public API lives only behind the CLI commands.
* Errors: wrap with `fmt.Errorf("…: %w", err)` so the chain survives.
* No global state beyond the cobra command tree and the build-time version vars.

## Commit conventions

We follow [Conventional Commits](https://www.conventionalcommits.org/) loosely:

```text
feat: add Avro decoder via schema registry
fix: handle empty header values in raw output
docs: clarify TUI key bindings
```

`feat:` and `fix:` show up automatically in goreleaser changelogs.

## Submitting a PR

1. Fork and branch off `main`.
2. Add tests for behavioral changes (`internal/format`, `internal/config` are good models).
3. Update the README if you change user-visible flags or add a command.
4. Open the PR with a brief description of *why* — the *what* is in the diff.

## Releasing (maintainers)

```sh
git tag v0.2.0
git push origin v0.2.0
```

The `release` workflow runs goreleaser, publishes archives to the GitHub release, and pushes multi-arch images to `ghcr.io/darioajr/kafko`.
