# source-udp-forwarder

[![github-actions](https://github.com/startersclan/source-udp-forwarder/workflows/ci/badge.svg)](https://github.com/startersclan/source-udp-forwarder/actions)
[![github-release](https://img.shields.io/github/v/release/startersclan/source-udp-forwarder?style=flat-square)](https://github.com/startersclan/source-udp-forwarder/releases/)
[![docker-image-size](https://img.shields.io/docker/image-size/startersclan/source-udp-forwarder/latest)](https://hub.docker.com/r/startersclan/source-udp-forwarder)
[![codecov](https://codecov.io/gh/startersclan/source-udp-forwarder/branch/master/graph/badge.svg)](https://codecov.io/gh/startersclan/source-udp-forwarder)
[![go-report-card](https://goreportcard.com/badge/github.com/startersclan/source-udp-forwarder)](https://goreportcard.com/report/github.com/startersclan/source-udp-forwarder)

A simple UDP forwarder to the HLStatsX:CE daemon.

## Agenda

The [HLStatsX:CE perl daemon](https://github.com/startersclan/hlstatsx-community-edition/tree/master/scripts) infers a gameserver's IP:PORT from the client socket from which it receives (reads) the gameserver's `logaddress_add` logs. This means both the daemon and the gameservers have to run on the same network.

This UDP forwarder eliminates this need by leveraging on an already built-in proxy protocol in the daemon - It simply runs as a sidecar to the gameserver, receives logs from the gameserver, prepends each log line with a spoofed `IP:PORT` as well as a secret [`PROXY_KEY`](https://github.com/startersclan/hlstatsx-community-edition/blob/v1.6.19/scripts/hlstats.pl#L1780) only known by the daemon, and finally sends that log line to the daemon. The daemon reads the gameserver's `IP:PORT` from each log line, rather than the usual inferring it from the client socket.

`source-udp-forwarder` uses less than `3MB` of memory.

## Install

### Binaries

Binaries are on the [releases](https://github.com/startersclan/source-udp-forwarder/releases/) page.

### Docker

```sh
docker run -it startersclan/source-udp-forwarder:latest
```

## Demo

1. Start the gameserver with cvar `logaddress_add 0.0.0.0:26999` for `srcds` (`srcds` refuses to log to `logaddress_add 127.0.0.1:<PORT>` for some reason) or `logaddress_add 127.0.0.1 26999` for `hlds` servers, to ensure the gameserver send logs to `source-udp-forwarder`.

2. Start `source-udp-forwarder` as a sidecar to the gameserver (both on localhost), setting the follow environment variables:

    - `UDP_FORWARD_ADDR` to the HLStatsX:CE perl daemon's IP:PORT or HOSTNAME:PORT
    - `FORWARD_PROXY_KEY` to the proxy key secret defined in HLStatsX:CE settings
    - `FORWARD_GAMESERVER_IP` to the gameserver's IP as registered in HLStatsX:CE database
    - `FORWARD_GAMESERVER_PORT` to the gameserver's PORT as registered in HLStatsX:CE database
    - `LOG_LEVEL` to `DEBUG` to ensure it's receiving logs from the gameserver. You can revert this back to `INFO` once everything is working.

3. Watch the daemon logs to ensure it's receiving logs from `source-udp-forwarder`. There should be a `PROXY` event tag attached to each log line received from `source-udp-forwarder`.

Here are some `docker-compose` examples demonstrating a gameserver UDP logs being proxied via `source-udp-forwarder` to the HLStatsX:CE perl daemon:

- [Counterstrike 1.6](docs/hlds-cstrike-example/docker-compose.yml) - This will work for all [GoldSource](https://developer.valvesoftware.com/wiki/GoldSrc) games, such as Half-Life and Condition Zero
- [Half-Life 2 multiplayer logs](docs/srcds-hl2mp-example/docker-compose.yml). This will work for all [Source](https://developer.valvesoftware.com/wiki/Source) games, such as Counter-Strike Global Offensive and Left 4 Dead 2.

## Usage

Configuration is done via (from highest priorty to lowest priority):

1. Command line
2. Environment variables

If `1.` and `2.` are used simultaneously, `1.` takes precedence.

### Command line

Run `source-udp-forwarder -help` to see command line usage:

### Environment variables

| Environment variable | Description |
|---|---|
| `UDP_LISTEN_ADDR`  | `<IP>:<PORT>` to listen on for incoming packets. Default value: `:26999` |
| `UDP_FORWARD_ADDR`  | `<IP>:<PORT>` or `<HOSTNAME>:<PORT>` to which incoming packets will be forwarded. Default value: `1.2.3.4:1013` |
| `FORWARD_PROXY_KEY`  | The [`PROXY_KEY`](https://github.com/startersclan/hlstatsx-community-edition/blob/v1.6.19/scripts/hlstats.pl#L1780) secret defined in the HLStatsX:CE Web Admin Panel. Default value: `XXXXX` |
| `FORWARD_GAMESERVER_IP`  | IP that the sent packet should include. Default value: `127.0.0.1` |
| `FORWARD_GAMESERVER_PORT`  | Port that the sent packet should include. Default value: `27015` |
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
