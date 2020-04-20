# Peer Calls v4 (alpha)

[![Build Status][travis-badge]][travis]

[travis-badge]: https://travis-ci.org/peer-calls/peer-calls.svg?branch=server-go
[travis]: https://travis-ci.org/peer-calls/peer-calls

WebRTC peer to peer calls for everyone. See it live in action at
[peercalls.com/alpha][peer-calls].

[peer-calls]: https://peercalls.com/alpha

This branch contains experimental work. The server has been completely rewriten
in Go and all the original functionality works. An optional implementation of a
Selective Forwarding Unit (SFU) is being made to make Peer Calls consume less
bandwith for user video uploads. Once implemented, it will be released as `Peer
Calls v4`.

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

## Using Docker

Use the [`peer-calls/peer-calls`][hub] image from Docker Hub:

```bash
docker run --rm -it -p 3000:3000 peer-calls/peer-calls:alpha
```

[hub]: https://hub.docker.com/r/peer-calls/peer-calls

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
docker run --rm -it -p 3000:3000 peer-calls:alpha
```

# Configuration

## Environment variables


| Variable                            | Type   | Description                                                                  | Default   |
|-------------------------------------|--------|------------------------------------------------------------------------------|-----------|
| `PEERCALLS_LOG`                     | csv    | Enables or disables logging for certain modules                              | `-sdp,-ws,-pion:*:trace,-pion:*:debug,-pion:*:info,*` |
| `PEERCALLS_BASE_URL`                | string | Base URL of the application                                                  |           |
| `PEERCALLS_BIND_HOST`               | string | IP to listen to                                                              | `0.0.0.0` |
| `PEERCALLS_BIND_PORT`               | int    | Port to listen to                                                            | `3000`    |
| `PEERCALLS_TLS_CERT`                | string | Path to TLS PEM certificate. If set will enable TLS                          |           |
| `PEERCALLS_TLS_KEY`                 | string | Path to TLS PEM cert key. If set will enable TLS                             |           |
| `PEERCALLS_STORE_TYPE`              | string | Can be `memory` or `redis`                                                   | `memory`  |
| `PEERCALLS_STORE_REDIS_HOST`        | string | Hostname of Redis server                                                     |           |
| `PEERCALLS_STORE_REDIS_PORT`        | int    | Port of Redis server                                                         |           |
| `PEERCALLS_STORE_REDIS_PREFIX`      | string | Prefix for Redis keys. Suggestion: `peercalls`                               |           |
| `PEERCALLS_NETWORK_TYPE`            | string | Can be `mesh` or `sfu`. Setting to SFU will make the server the main peer    | `mesh`    |
| `PEERCALLS_NETWORK_SFU_INTERFACES`  | csv    | List of interfaces to use for ICE candidates, uses all available when empty  |           |
| `PEERCALLS_ICE_SERVER_URLS`         | csv    | List of ICE Server URLs                                                      |           |
| `PEERCALLS_ICE_SERVER_AUTH_TYPE`    | string | Can be empty or `secret` for coturn `static-auth-secret` config option.      |           |
| `PEERCALLS_ICE_SERVER_SECRET`       | string | Secret for coturn                                                            |           |
| `PEERCALLS_ICE_SERVER_USERNAME`     | string | Username for coturn                                                          |           |

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
```

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

Below are some common NPM scripts that are used for development:

```
npm start              start the precompiled server.
npm run build          build all client-side resources.
npm run start:server   start and compile server-side TypeScript on the fly,
                       restarts the server when the resources change.
npm run start:watch    start the server, and recompile client-side resources
                       when the sources change.
npm test               run all tests.
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
static-auth-secret=PASSWORD
realm=example.com
total-quota=300
cert=/etc/letsencrypt/live/rtc.example.com/fullchain.pem
pkey=/etc/letsencrypt/live/rtc.example.com/privkey.pem
log-file=/dev/stdout
no-multicast-peers
proc-user=turnserver
proc-group=turnserver
```

Change the `PASSWORD`, `realm`  and paths to server certificates.

Use the following configuration for Peer Calls:

```yaml
iceServers:
- url: 'turn:rtc.example.com'
  urls: 'turn:rtc.example.com'
  username: 'example'
  secret: 'PASSWORD'
  auth: 'secret'
```

Finally, enable and start the `coturn` service:

```bash
sudo systemctl enable coturn
sudo systemctl start coturn
```

# TODO

- [x] Do not require config files and allow configuration via environment
  variables. (Fixed in 23fabb0)
- [x] Show menu dialog before connecting (Fixed in 0b4aa45)
- [x] Reduce production build size by removing Pug. (Fixed in 2d14e5f c743f19)
- [x] Add ability to share files (Fixed in 3877893)
- [x] Add Socket.IO support for Redis (to scale horizontally).
- [x] Allow other methods of connectivity, beside mesh.
- [ ] Fix connectivity issues with SFU

# Contributing

See [Contributing](CONTRIBUTING.md) section.

If you encounter a bug, please open a new issue! Thank you ❤️

# License

[Apache 2.0](LICENSE)
