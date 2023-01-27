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
- [x] Add handling of other RTCP packets besides NACK, PLI and REMB
- [x] Add JitterBuffer (experimental, currently without congestion control)
- [x] Support multiple Peer Calls nodes when using SFU
- [x] Add support for passive ICE TCP candidates
- [x] End-to-End Encryption (E2EE) using Insertable Streams. See [#142](https://github.com/peer-calls/peer-calls/pull/142).

# Requirements for Development

 - [Node.js 18.13][node]
 - [Go 1.19.5][go]

Alternatively, Docker  can be used to run Peer Calls.

[node]: https://nodejs.org
[go]: https://golang.org/

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

## Deploying onto Kubernetes

The root of this repository contains a `kustomization.yaml`, allowing anyone to
patch the manifests found within the `deploy/` directory. To deploy the manifests
without applying any patches, pass the URL to `kubectl`:

```bash
kubectl apply -k github.com/peer-calls/peer-calls
```

## Using Docker

The automated builds on Docker Hub now require a subscription, and approval is
required even for open source projects. We recently switched to using GitHub
Container Registry instead:

Use the [`ghcr.io/peer-calls/peer-calls`][ghcr] image:

```bash
docker run --rm -it -p 3000:3000 ghcr.io/peer-calls/peer-calls:latest
```

[ghcr]: https://ghcr.io/peer-calls/peer-calls

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
| `PEERCALLS_FS`                       | string | When set to a non-empty value, use the path to find resource files           |           |
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
| `PEERCALLS_NETWORK_SFU_PROTOCOLS`    | csv    | Can be `udp4`, `udp6`, `tcp4` or `tcp6`                                      | `udp4,udp6` |
| `PEERCALLS_NETWORK_SFU_TCP_BIND_ADDR`| string | ICE TCP bind address. By default listens on all interfaces.                  |           |
| `PEERCALLS_NETWORK_SFU_TCP_LISTEN_PORT`| int  | ICE TCP listen port. By default uses a random port.                          | `0`       |
| `PEERCALLS_NETWORK_SFU_TRANSPORT_LISTEN_ADDR` | string | When set, will listen for external RTP, Data and Metadata UDP streams |           |
| `PEERCALLS_NETWORK_SFU_TRANSPORT_NODES`| csv    | When set, will transmit media and data to designated `host:port`(s).  |           |
| `PEERCALLS_NETWORK_SFU_UDP_PORT_MIN` | int    | Defines ICE UDP range start to use for UDP host candidates.                  | `0`       |
| `PEERCALLS_NETWORK_SFU_UDP_PORT_MAX` | int    | Defines ICE UDP range end to use for UDP host candidates.                    | `0`       |
| `PEERCALLS_ICE_SERVER_URLS`          | csv    | List of ICE Server URLs                                                      |           |
| `PEERCALLS_ICE_SERVER_AUTH_TYPE`     | string | Can be empty or `secret` for coturn `static-auth-secret` config option.      |           |
| `PEERCALLS_ICE_SERVER_SECRET`        | string | Secret for coturn                                                            |           |
| `PEERCALLS_ICE_SERVER_USERNAME`      | string | Username for coturn                                                          |           |
| `PEERCALLS_PROMETHEUS_ACCESS_TOKEN`  | string | Access token for prometheus `/metrics` URL                                   |           |
| `PEERCALLS_FRONTEND_ENCODED_INSERTABLE_STREAMS` | bool | Enable insertable streams                                           | `false`   |

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
frontend:
  encodedInsertableStreams: false
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

# Epheremal UDP Ports for ICE

The UDP port range can be defined for opening epheremal ports. These ports will
be used for generating UDP host ICE candidates. It is recommended to enable
these UDP ports when ICE TCP is enabled, because the priority of TCP host
candidates will be higher than srflx/prflx candidates, as such TCP will be used
even though UDP connectivity might be possible.

# ICE TCP

Peer Calls supports ICE over TCP as described in RFC6544. Currently only
passive ICE candidates are supported. This means that users whose ISPs or
corporate firewalls block UDP packets can use TCP to connect to the SFU. In
most scenarios, this removes the need to use a TURN server, but this
functionality is currently experimental and is not enabled by default.

Add the `tcp4` and `tcp6` to your `PEERCALLS_NETWORK_SFU_PROTOCOLS` to enable
support for ICE TCP:

```
PEERCALLS_NETWORK_TYPE=sfu PEERCALLS_NETWORK_SFU_PROTOCOLS=`udp4,udp6,tcp4,tcp6` peer-calls
```

To test this functionality, `udp4` and `udp6` network types should be omitted:

```
PEERCALLS_NETWORK_TYPE=sfu PEERCALLS_NETWORK_SFU_PROTOCOLS=`tcp4,tcp6` peer-calls
```

Please note that in production the `PEERCALLS_NETWORK_SFU_TCP_LISTEN_PORT` should
be specified and external TCP access allowed through the server firewall.

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

# License

[Apache 2.0](LICENSE)
