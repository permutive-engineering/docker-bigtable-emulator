# docker-bigtable-emulator

Provides a multi-platform (`linux/amd64` and `linux/arm64`) Docker image of the Cloud Bigtable emulator, for local and
CI testing.

```
ghcr.io/permutive-engineering/bigtable-emulator:0.1.0
```

The image runs the in-memory emulator from [`cloud.google.com/go/bigtable/bttest`][bttest], the same one `gcloud beta
emulators bigtable start` runs. The version of that library is in `go.mod`. The image is rebuilt every Monday to pick
up updates to its base image, and pushed on every merge to `main`.

The tag is the version in `VERSION`. Bump the patch for a library update that changes nothing for users, and the minor
for a change to the environment variables, the schema format or when the port opens.

It creates the tables in `BIGTABLE_SCHEMA` before it opens its port. A client that can connect always finds them, so
`nc -z localhost 8086` is a correct healthcheck.

## Running it

```shell
docker run --rm -p 8086:8086 \
  -e BIGTABLE_PROJECT=my-project \
  -e BIGTABLE_INSTANCE=my-instance \
  -e BIGTABLE_SCHEMA="users:profile#settings,events:data" \
  ghcr.io/permutive-engineering/bigtable-emulator:0.1.0
```

| Variable                 | Default        | Description                                                         |
|:-------------------------|:---------------|:--------------------------------------------------------------------|
| `BIGTABLE_PROJECT`       | `dev`          | Project the tables are created in.                                  |
| `BIGTABLE_INSTANCE`      | `dev`          | Instance the tables are created in.                                 |
| `BIGTABLE_SCHEMA`        |                | Tables and column families: `<table>:<family>#<family>,<table>:...` |
| `BIGTABLE_EMULATOR_HOST` | `0.0.0.0:8086` | Address the emulator listens on.                                    |

Point a client at it with `BIGTABLE_EMULATOR_HOST=localhost:8086`.

## Building it locally

For your machine's platform:

```shell
docker build -t bigtable-emulator .
```

For both platforms, as CI does, you need a builder that uses the `docker-container` driver:

```shell
docker buildx create --name multi-platform --driver docker-container
docker buildx build --builder multi-platform --platform linux/amd64,linux/arm64 -t bigtable-emulator .
```

[bttest]: https://pkg.go.dev/cloud.google.com/go/bigtable/bttest
