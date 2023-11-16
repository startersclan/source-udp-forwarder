# source-udp-forwarder

[![github-actions](https://github.com/startersclan/source-udp-forwarder/workflows/ci/badge.svg)](https://github.com/startersclan/source-udp-forwarder/actions)
[![github-release](https://img.shields.io/github/v/release/startersclan/source-udp-forwarder?style=flat-square)](https://github.com/startersclan/source-udp-forwarder/releases/)
[![docker-image-size](https://img.shields.io/docker/image-size/startersclan/source-udp-forwarder/latest)](https://hub.docker.com/r/startersclan/source-udp-forwarder)
[![codecov](https://codecov.io/gh/startersclan/source-udp-forwarder/branch/master/graph/badge.svg)](https://codecov.io/gh/startersclan/source-udp-forwarder)
[![go-report-card](https://goreportcard.com/badge/github.com/startersclan/source-udp-forwarder)](https://goreportcard.com/report/github.com/startersclan/source-udp-forwarder)

A simple HTTP and UDP log forwarder to the [HLStatsX:CE daemon](https://github.com/startersclan/hlstatsx-community-edition).

## Agenda

The [HLStatsX:CE perl daemon](https://github.com/startersclan/hlstatsx-community-edition/tree/master/scripts) infers a gameserver's IP:PORT from the client socket from which it receives (reads) the gameserver's `logaddress_add_http` or `logaddress_add` logs. This means both the daemon and the gameservers have to run on the same network.

This log forwarder eliminates this need by leveraging on an already built-in proxy protocol in the daemon - It simply runs as a sidecar to the gameserver, receives logs from the gameserver, prepends each log line with a spoofed `IP:PORT` as well as a [`proxy_key`](https://github.com/startersclan/hlstatsx-community-edition/blob/1.6.19/scripts/hlstats.pl#L1780) secret only known by the daemon, and finally sends that log line to the daemon. The daemon reads the gameserver's `IP:PORT` from each log line, rather than the usual inferring it from the client socket.

`source-udp-forwarder` uses less than `3MB` of memory.

## Usage

### Binaries

Binaries are on the [releases](https://github.com/startersclan/source-udp-forwarder/releases/) page.

### Docker

Docker images are available on [Docker Hub](https://hub.docker.com/r/startersclan/source-udp-forwarder).

To run the latest stable version:

```sh
docker run -it startersclan/source-udp-forwarder:latest
```

To run a specific version, for example `v0.3.0`:

```sh
docker run -it startersclan/source-udp-forwarder:v0.3.0
```

## Demo

1. Start the gameserver with cvar `logaddress_add_http "http://127.0.0.1:26999"` for Counter-Strike 2, `logaddress_add 0.0.0.0:26999` for `srcds` (`srcds` refuses to log to `logaddress_add 127.0.0.1:<PORT>` for some reason), or `logaddress_add 127.0.0.1 26999` for `hlds` servers, and cvar `log on`, to ensure the gameserver send logs to `source-udp-forwarder`.

2. Start `source-udp-forwarder` as a sidecar to the gameserver (both on localhost), setting the follow environment variables:

    - `UDP_FORWARD_ADDR` to the HLStatsX:CE perl daemon's IP:PORT or HOSTNAME:PORT
    - `FORWARD_PROXY_KEY` to the proxy key secret defined in HLStatsX:CE settings
    - `FORWARD_GAMESERVER_IP` to the gameserver's IP as registered in HLStatsX:CE database
    - `FORWARD_GAMESERVER_PORT` to the gameserver's PORT as registered in HLStatsX:CE database
    - `LOG_LEVEL` to `DEBUG` to ensure it's receiving logs from the gameserver. You can revert this back to `INFO` once everything is working.

3. Watch the daemon logs to ensure it's receiving logs from `source-udp-forwarder`. There should be a `PROXY` event tag attached to each log line received from `source-udp-forwarder`.

See `docker-compose` examples:

- [Counter-Strike 2](docs/srcds-cs2-example/docker-compose.yml) - Works for Counter-Strike 2 and all games that sends logs using HTTP
- [Counter-Strike 1.6](docs/hlds-cstrike-example/docker-compose.yml) - This will work for all [GoldSource](https://developer.valvesoftware.com/wiki/GoldSrc) games which sends logs using UDP, such as Half-Life and Condition Zero
- [Half-Life 2 Multiplayer](docs/srcds-hl2mp-example/docker-compose.yml). This will work for all [Source](https://developer.valvesoftware.com/wiki/Source) games which sends logs using UDP, such as Counter-Strike Global Offensive and Left 4 Dead 2.

## Configuration

Configuration is done via (from highest to lowest priority):

1. Command line
2. Environment variables

If `1.` and `2.` are used simultaneously, `1.` takes precedence.

### Command line

Run `source-udp-forwarder -help` to see command line usage:

### Environment variables

| Environment variable | Description |
|---|---|
| `LISTEN_ADDR` | `<IP>:<PORT>` to listen for incoming HTTP and UDP logs. Default value: `:26999` |
| `UDP_FORWARD_ADDR` | `<IP>:<PORT>` of the `daemon` to which incoming packets will be forwarded. Default value: `127.0.0.1:27500` |
| `FORWARD_PROXY_KEY` | The [`proxy_key`](https://github.com/startersclan/hlstatsx-community-edition/blob/1.6.19/scripts/hlstats.pl#L1780) secret defined in the HLStatsX:CE Web Admin Panel. Default value: `XXXXX` |
| `FORWARD_GAMESERVER_IP` | IP that the sent packet should include. Default value: `127.0.0.1` |
| `FORWARD_GAMESERVER_PORT` | Port that the sent packet should include. Default value: `27015` |
| `LOG_LEVEL` | Log level. Defaults to `INFO`. May be one of the following (starting with the most verbose): `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`. Default value: `INFO`|
| `LOG_FORMAT` | Log format, valid options are `txt` and `json`. Default value: `txt` |

## Development

Requires `make`, `docker`, and `docker-compose` if you want all `make` commands to be working.

Requires [`go`](https://golang.org/doc/install) only if you are developing.

```sh
# Print usage
make help

# Build
make build # Defaults to linux amd64
make build GOOS=linux GOARCH=arm64 # For arm64

# Build docker image
make build-image # Defaults to linux amd64
make build-image GOOS=linux GOARCH=arm64 # For arm64

# Build multiarch docker images
make buildx-image # Build
make buildx-image REGISTRY=xxx REGISTRY_USER=xxx BUILDX_PUSH=true BUILDX_TAG_LATEST=true # Build and push

# Start a shell in a container
make shell

# Test
make test

# Cleanup
make clean
```
