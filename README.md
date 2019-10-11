# source-udp-forwarder

[dockerhub-badge]: https://img.shields.io/badge/docker%20hub-sourceservers-blue.svg?logo=docker&logoColor=2596EC&color=FFA722&label=&labelColor=&style=popout-square

A simple UDP forwarder to the HLStatsX:CE daemon.

## Agenda

The HLStatsXCE:CE daemon infers a gameserver's IP:PORT from the client socket from which it receives (reads) the gameserver's `logaddress_add` logs. This means both the daemon and the gameservers have to run on the same network.

Put simply, this UDP forwarder eliminates this need by leveraging on the an already built-in proxy protocol in the daemon - It simply run as a sidecar to the gameserver, receives logs from the gameserver, prepends each log line with a spoofed IP:PORT as well as a secret PROXY_KEY only known by the daemon, and finally sends that log line to the daemon. The daemon reads the gameserver's IP:PORT from each log line, rather than the usual inferring it from the client socket.

## How to use

1. Start the gameserver with `logaddress_add 127.0.0.1:26999` for `srcds` or `logaddress_add 127.0.0.1 26999` for `goldsource` servers, to ensure the gameserver send logs to `source-udp-forwarder`.

2. Start `source-udp-forwarder` as a sidecar to the gameserver (both on localhost), setting the follow environment variables:

- `UDP_FORWARD_ADDR` to the daemon's listening IP:PORT

- `FORWARD_PROXY_KEY` to the proxy key secret defined in HLStatsX:CE settings

- `FORWARD_GAMESERVER_IP` to the gameserver's IP as registered in HLStatsX:CE database

- `FORWARD_GAMESERVER_PORT` to the gameserver's PORT as registered in HLStatsX:CE database

- `LOG_LEVEL` to `DEBUG` to ensure it's receiving logs from the gameserver. You can revert this back to `INFO` once everything is working.

3. Watch the daemon logs to ensure it's receiving logs from `source-udp-forwarder`. There should be a `PROXY` event tag attached to each log line received from `source-udp-forwarder`.

## Environment variables

| Environment variable | Description |
|---|---|
| `UDP_LISTEN_ADDR`  | `<IP>:<Port>` to listen on for incoming packets. Default value: `:26999` |
| `UDP_FORWARD_ADDR`  | `<IP>:<Port>` to which incoming packets will be forwarded. Default value: `1.2.3.4:1013` |
| `FORWARD_PROXY_KEY`  | The PROXY_KEY secret defined in HLStatsX:CE settings. Default value: `XXXXX` |
| `FORWARD_GAMESERVER_IP`  | IP that the sent packet should include. Default value: `127.0.0.1` |
| `FORWARD_GAMESERVER_PORT`  | "Port that the sent packet should include. Default value: `27015` |
| `LOG_LEVEL` | Log level. Defaults to `INFO`. May be one of the following (starting with the most verbose): `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`. Default value: `INFO`|
| `LOG_FORMAT` | Log format, valid options are `txt` and `json`. Default value: `txt` |
