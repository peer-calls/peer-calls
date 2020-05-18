# Peer Calls v4

![Peer Calls CI](https://github.com/peer-calls/peer-calls/workflows/Peer%20Calls%20CI/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/peer-calls/peer-calls)](https://goreportcard.com/report/github.com/peer-calls/peer-calls)

WebRTC peer to peer calls for everyone. See it live in action at
[peercalls.com][peer-calls].

[peer-calls]: https://peercalls.com

The server has been completely rewriten in Go and all the original
functionality works. An optional implementation of a Selective Forwarding Unit
(SFU) is available to make Peer Calls consume less bandwith for user video
uploads. This wouldn't haven been possible without the awesome
[pion/webrtc][pion] library.

[pion]: https://github.com/pion/webrtc

The config file format is still YAML, but is different than what was in v3. The
v3 source code is available in `version-3` branch.  Version 4 will no longer be
published on NPM since the server is no longer written in NodeJS.

# What's New in v4

- [x] Core rewritten in Golang.
- [x] Selective Forwarding Unit. Can be enabled using `NETWORK_TYPE=sfu` environment variable. The [peercalls.com][peer-calls] instance has this enabled.
- [x] Ability to change video and audio devices without reconnecting.
- [x] Improved toolbar layout. Can be toggled by clicking or tapping.
- [x] Multiple videos are now shown in a full-size grid and each can be minimized.
- [x] Video cropping can be turned off.
- [x] Improved file sending. Users are now able to send files larger than 64 or 256 KB (depends on the browser).
- [x] Device names are correctly populated in the dropdown list.
- [x] Improved desktop sharing.
- [x] Copy invite link to clipboard. Will show as share icon on devices that support it.
- [x] Fix: Toolbar icons render correctly on iOS 12 devices.
- [x] Fix: Video autoplays.
- [x] Fix: Toolbar is no longer visible until call is joined
- [x] Fix: Add warning when using an unsupported browser
- [x] Fix: Add warning when JavaScript is disabled

## TODO for Selective Forwarding Unit

- [x] Support dynamic adding and removing of streams
- [x] Support RTCP packet Picture Loss Indicator (PLI)
- [x] Support RTCP packet Receiver Estimated Maximum Bitrate (REMB)
- [ ] Add handling of other RTCP packets besides NACK, PLI and REMB
- [x] Add JitterBuffer (experimental, currently without congestion control)
- [ ] Support multiple Peer Calls nodes when using SFU

# Requirements

 - [Node.js 8][node] or [Node.js 12][node]
 - [Go 1.14][go]

Alternatively, [Docker][docker] can be used to run Peer Calls.

[node]: https://nodejs.org
[go]: https://golang.org/
[docker]: https://www.docker.com/

# Stack

## Backend

 - [Golang][go]
 - [pion/webrtc][pion]
 - github.com/go-chi/chi
 - nhooyr.io/websocket

[pion]: https://github.com/pion/webrtc

See [go.mod](go.mod) for more information

## Frontend

 - React
 - Redux
 - TypeScript (since peer-calls `v2.1.0`)

See [package.json](package.json) for more information.

# Installation & Running

## Download Release

Head to [Releases](https://github.com/peer-calls/peer-calls/releases) and
download a precompiled version. Currently the binaries for the following
systems are built automatically:

 - linux amd64
 - linux arm
 - darwin (macOS) amd64
 - windows amd64

## Using Docker

Use the [`peercalls/peercalls`][hub] image from Docker Hub:

```bash
docker run --rm -it -p 3000:3000 peercalls/peercalls:latest
```

[hub]: https://hub.docker.com/r/peercalls/peercalls

## Building from Source

```bash
git clone https://github.com/peer-calls/peer-calls.git
cd peer-calls
npm install

# for production
npm run build
npm run build:go:linux

# for development
npm run start
```

## Building Docker Image

```bash
git clone https://github.com/peer-calls/peer-calls
cd peer-calls
docker build -t peer-calls .
docker run --rm -it -p 3000:3000 peer-calls
```

# Configuration

## Environment variables


| Variable                             | Type   | Description                                                                  | Default   |
|--------------------------------------|--------|------------------------------------------------------------------------------|-----------|
| `PEERCALLS_LOG`                      | csv    | Enables or disables logging for certain modules                              | `-sdp,-ws,-nack,-rtp,-rtcp,-pion:*:trace,-pion:*:debug,-pion:*:info,*` |
| `PEERCALLS_BASE_URL`                 | string | Base URL of the application                                                  |           |
| `PEERCALLS_BIND_HOST`                | string | IP to listen to                                                              | `0.0.0.0` |
| `PEERCALLS_BIND_PORT`                | int    | Port to listen to                                                            | `3000`    |
| `PEERCALLS_TLS_CERT`                 | string | Path to TLS PEM certificate. If set will enable TLS                          |           |
| `PEERCALLS_TLS_KEY`                  | string | Path to TLS PEM cert key. If set will enable TLS                             |           |
| `PEERCALLS_STORE_TYPE`               | string | Can be `memory` or `redis`                                                   | `memory`  |
| `PEERCALLS_STORE_REDIS_HOST`         | string | Hostname of Redis server                                                     |           |
| `PEERCALLS_STORE_REDIS_PORT`         | int    | Port of Redis server                                                         |           |
| `PEERCALLS_STORE_REDIS_PREFIX`       | string | Prefix for Redis keys. Suggestion: `peercalls`                               |           |
| `PEERCALLS_NETWORK_TYPE`             | string | Can be `mesh` or `sfu`. Setting to SFU will make the server the main peer    | `mesh`    |
| `PEERCALLS_NETWORK_SFU_INTERFACES`   | csv    | List of interfaces to use for ICE candidates, uses all available when empty  |           |
| `PEERCALLS_NETWORK_SFU_JITTER_BUFFER`| bool   | Set to `true` to enable the use of Jitter Buffer                             | `false`   |
| `PEERCALLS_ICE_SERVER_URLS`          | csv    | List of ICE Server URLs                                                      |           |
| `PEERCALLS_ICE_SERVER_AUTH_TYPE`     | string | Can be empty or `secret` for coturn `static-auth-secret` config option.      |           |
| `PEERCALLS_ICE_SERVER_SECRET`        | string | Secret for coturn                                                            |           |
| `PEERCALLS_ICE_SERVER_USERNAME`      | string | Username for coturn                                                          |           |
| `PEERCALLS_PROMETHEUS_ACCESS_TOKEN`  | string | Access token for prometheus `/metrics` URL                                   |           |

The default ICE servers in use are:

- `stun:stun.l.google.com:19302`
- `stun:global.stun.twilio.com:3478?transport=udp`

Only a single ICE server can be defined via environment variables. To define
more use a YAML config file. To load a config file, use the `-c
/path/to/config.yml` command line argument.

See [config/types.go][config] for configuration types.

[config]: ./src/server/config/types.go

Example:

```yaml
base_url: ''
bind_host: '0.0.0.0'
bind_port: 3005
ice_servers:
 - urls:
   - 'stun:stun.l.google.com:19302'
- urls:
  - 'stun:global.stun.twilio.com:3478?transport=udp'
#- urls:
#  - 'turn:coturn.mydomain.com'
#  auth_type: secret
#  auth_secret:
#    username: "peercalls"
#    secret: "some-static-secret"
# tls:
#   cert: test.pem
#   key: test.key
store:
  type: memory
  # type: redis
  # redis:
  #   host: localhost
  #   port: 6379
  #   prefix: peercalls
network:
  type: mesh
  # type: sfu
  # sfu:
  #   interfaces:
  #   - eth0
prometheus:
  access_token: "mytoken"
```

Prometheus `/metrics` URL will not be accessible without an access token set.
The access token can be provided by either:

- Setting `Authorization` header to `Bearer mytoken`, or
- Providing the access token as a query string: `/metrics?access_token=mytoken`

To access the server, go to http://localhost:3000.

# Accessing From Network

Most browsers will prevent access to user media devices if the application is
accessed from the network (not via 127.0.0.1). If you wish to test your mobile
devices, you'll have to enable TLS by setting the `PEERCALLS_TLS_CERT` and
`PEERCALLS_TLS_KEY` environment variables. To generate a self-signed certificate
you can use:

```
openssl req -nodes -x509 -newkey rsa:4096 -keyout key.pem -subj "/C=US/ST=Oregon/L=Portland/O=Company Name/OU=Org/CN=example.com" -out cert.pem -days 365
```

Replace `example.com` with your server's hostname.

# Multiple Instances and Redis

Redis can be used to allow users connected to different instances to connect.
The following needs to be added to `config.yaml` to enable Redis:

```yaml
store:
  type: redis
  redis:
    host: redis-host  # redis host
    port: 6379        # redis port
    prefix: peercalls # all instances must use the same prefix
```

# Logging

By default, Peer Calls server will log only basic information. Client-side
logging is disabled by default.

Server-side logs can be configured via the `PEERCALLS_LOG` environment variable. Setting
it to `*` will enable all server-side logging:

- `PEERCALLS_LOG=*`

Client-side logs can be configured via `localStorage.DEBUG` and
`localStorage.LOG` variables:

- Setting `localStorage.log=1` enables logging of Redux actions and state
  changes
- Setting `localStorage.debug=peercalls,peercalls:*` enables all other
  client-side logging

# Development

Below are some common scripts used for development:

```
npm start              build all resources and start the server.
npm run build          build all client-side resources.
npm run start:server   start the server
npm run js:watch       build and watch resources
npm test               run all client-side tests.
go test ./...          run all server tests
npm run ci             run all linting, tests and build the client-side
```

# Browser Support

Tested on Firefox and Chrome, including mobile versions. Also works on Safari
and iOS since version 11. Does not work on Microsoft Edge because they do not
support DataChannels yet.

For more details, see here:

- http://caniuse.com/#feat=rtcpeerconnection
- http://caniuse.com/#search=getUserMedia

In Firefox, it might be useful to use `about:webrtc` to debug connection
issues. In Chrome, use `about:webrtc-internals`.

When experiencing connection issues, the first thing to try is to have all
peers to use the same browser.

# TURN Server

When a direct connection cannot be established, it might be help to use a TURN
server. The peercalls.com instance is configured to use a TURN server and it
can be used for testing. However, the server bandwidth there is not unlimited.

Here are the steps to install a TURN server on Ubuntu/Debian Linux:

```bash
sudo apt install coturn
```

Use the following configuration as a template for `/etc/turnserver.conf`:

```bash
lt-cred-mech
use-auth-secret
static-auth-secret=p4ssw0rd
realm=example.com
total-quota=300
cert=/etc/letsencrypt/live/rtc.example.com/fullchain.pem
pkey=/etc/letsencrypt/live/rtc.example.com/privkey.pem
log-file=/dev/stdout
no-multicast-peers
proc-user=turnserver
proc-group=turnserver
```

Change the `p4ssw0rd`, `realm`  and paths to server certificates.

Use the following configuration for Peer Calls:

```yaml
iceServers:
- urls:
  - 'turn:rtc.example.com'
  auth_type: secret
  auth_secret:
    username: 'example'
    secret: 'p4ssw0rd'
```

Finally, enable and start the `coturn` service:

```bash
sudo systemctl enable coturn
sudo systemctl start coturn
```

# Contributing

See [Contributing](CONTRIBUTING.md) section.

If you encounter a bug, please open a new issue!

# Support

The development of Peer Calls is sponsored by [rondomoon][rondomoon]. If you'd
like enterprise on-site support or become a sponsor, please contact
[hello@rondomoon.com](mailto:hello@rondomoon.com).

[rondomoon]: https://rondomoon.com

If you wish to support future development of Peer Calls, you can donate here:

[![Donate](https://www.paypalobjects.com/en_US/i/btn/btn_donateCC_LG.gif)](https://www.paypal.com/cgi-bin/webscr?cmd=_s-xclick&hosted_button_id=364CXPNDPK2YG&source=url)

Thank you ❤️

# License

[Apache 2.0](LICENSE)
