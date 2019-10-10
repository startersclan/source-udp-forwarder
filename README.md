# source-udp-forwarder

A simple UDP forwarder to the HLStatsX:CE daemon.

## Agenda

The HLStatsXCE:CE daemon infers a gameserver's IP:PORT from the client socket from which it receives (read) the gameserver's `logaddress_add` logs. This means both the daemon and the gameservers have to run on the same network.

This UDP forwarder eliminates this need by leveraging on the an already built-in proxy protocol in the daemon - It simply forwards each log line prepended with a spoofed IP:PORT, along with a secret PROXY_KEY only known by the daemon. The daemon then uses IP:PORT from the log line, rather than inferring it from the client socket.

## Environment variables

| Environment variable | Description |
|---|---|
| `UDP_LISTEN_ADDR`  | `<IP>:<Port>` to listen on for incoming packets. |
| `UDP_FORWARD_ADDR`  | `<IP>:<Port>` to which incoming packets will be forwarded. |
| `FORWARD_PROXY_KEY`  | The PROXY_KEY secret defined in HLStatsX:CE settings. |
| `FORWARD_GAMESERVER_IP`  | IP that the sent packet should include. |
| `FORWARD_GAMESERVER_PORT`  | "Port that the sent packet should include. |
| `LOG_LEVEL` | Log level. Defaults to `INFO`. May be one of the following (starting with the most verbose): `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL` |
| `LOG_FORMAT` | Log format, valid options are `txt` and `json`. |
