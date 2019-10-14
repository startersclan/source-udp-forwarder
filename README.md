# source-udp-forwarder

[![travis-ci](https://img.shields.io/travis/startersclan/source-udp-forwarder/master)](https://travis-ci.org/startersclan/source-udp-forwarder)
[![docker-image-size](https://img.shields.io/microbadger/image-size/startersclan/source-udp-forwarder/latest)](https://hub.docker.com/r/startersclan/source-udp-forwarder)
[![docker-image-layers](https://img.shields.io/microbadger/layers/startersclan/source-udp-forwarder/latest)](https://hub.docker.com/r/startersclan/source-udp-forwarder)

A simple UDP forwarder to the HLStatsX:CE daemon.

## Agenda

The HLStatsXCE:CE daemon infers a gameserver's IP:PORT from the client socket from which it receives (reads) the gameserver's `logaddress_add` logs. This means both the daemon and the gameservers have to run on the same network.

Put simply, this UDP forwarder eliminates this need by leveraging on the an already built-in proxy protocol in the daemon - It simply run as a sidecar to the gameserver, receives logs from the gameserver, prepends each log line with a spoofed IP:PORT as well as a secret PROXY_KEY only known by the daemon, and finally sends that log line to the daemon. The daemon reads the gameserver's IP:PORT from each log line, rather than the usual inferring it from the client socket.

`source-udp-forwarder` uses less than `3MB` of memory.

## How to use

1. Start the gameserver with `logaddress_add 0.0.0.0:26999` for `srcds` (`srcds` refuses to log to `logaddress_add 127.0.0.1:<PORT>` for some reason) or `logaddress_add 127.0.0.1 26999` for `hlds` servers, to ensure the gameserver send logs to `source-udp-forwarder`.

2. Start `source-udp-forwarder` as a sidecar to the gameserver (both on localhost), setting the follow environment variables:

    - `UDP_FORWARD_ADDR` to the HLStatsX:CE daemon's IP:PORT or HOSTNAME:PORT

    - `FORWARD_PROXY_KEY` to the proxy key secret defined in HLStatsX:CE settings

    - `FORWARD_GAMESERVER_IP` to the gameserver's IP as registered in HLStatsX:CE database

    - `FORWARD_GAMESERVER_PORT` to the gameserver's PORT as registered in HLStatsX:CE database

    - `LOG_LEVEL` to `DEBUG` to ensure it's receiving logs from the gameserver. You can revert this back to `INFO` once everything is working.

3. Watch the daemon logs to ensure it's receiving logs from `source-udp-forwarder`. There should be a `PROXY` event tag attached to each log line received from `source-udp-forwarder`.

## Docker image

```sh
docker run -it startersclan/source-udp-forwarder
```

## Environment variables

| Environment variable | Description |
|---|---|
| `UDP_LISTEN_ADDR`  | `<IP>:<PORT>` to listen on for incoming packets. Default value: `:26999` |
| `UDP_FORWARD_ADDR`  | `<IP>:<PORT>` or `<HOSTNAME>:<PORT>` to which incoming packets will be forwarded. Default value: `1.2.3.4:1013` |
| `FORWARD_PROXY_KEY`  | The PROXY_KEY secret defined in HLStatsX:CE settings. Default value: `XXXXX` |
| `FORWARD_GAMESERVER_IP`  | IP that the sent packet should include. Default value: `127.0.0.1` |
| `FORWARD_GAMESERVER_PORT`  | Port that the sent packet should include. Default value: `27015` |
| `LOG_LEVEL` | Log level. Defaults to `INFO`. May be one of the following (starting with the most verbose): `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`. Default value: `INFO`|
| `LOG_FORMAT` | Log format, valid options are `txt` and `json`. Default value: `txt` |

## Development

Requires [`go`](https://golang.org/doc/install), `make`, `docker`, and `docker-compose` if you want all `make` commands to be working.

#### Mount a ramdisk on `./bin`

```sh
make mount-ramdisk
```

#### Build

```sh
make build  # Defaults to linux amd64
GOOS=linux GOARCH=arm64 make build # For arm64
# etc...
```

#### Build and run

```sh
make up
```

#### Build docker image

```sh
make build-image
```

#### Test

```sh
make test
```

#### Clean

```sh
make clean
```

#### Unmount ramdisk on `./bin`

```sh
make unmount-ramdisk
```
