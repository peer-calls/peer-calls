# Peer Calls

[![Build Status][travis-badge]][travis]
[![NPM Package][npm-badge]][npm]

[travis-badge]: https://travis-ci.org/jeremija/peer-calls.svg?branch=master
[travis]: https://travis-ci.org/jeremija/peer-calls
[npm-badge]: https://img.shields.io/npm/v/peer-calls.svg
[npm]: https://www.npmjs.com/package/peer-calls

WebRTC peer to peer calls for everyone. See it live in action at
[peercalls.com][peer-calls].

[peer-calls]: https://peercalls.com

Work in progress.

# Requirements

 - [Node.js 8][node], or
 - [Node.js 12][node], or
 - [Docker][docker]

[node]: https://nodejs.org
[docker]: https://www.docker.com/

# Stack

 - Express
 - Socket.IO
 - React
 - Redux
 - TypeScript (since peer-calls `v2.1.0`)

# Installation & Running

## Using npx (from NPM)

```bash
npx peer-calls
```

## Installing locally

```bash
npm install peer-calls
./node_modules/.bin/peer-calls
```

## Installing Globally

```bash
npm install --global peer-calls
peer-calls
```

## Using Docker

Use the [`jeremija/peer-calls`][hub] image from Docker Hub:

```bash
docker pull jeremija/peer-calls
docker run --rm -it -p 3000:3000 jeremija/peer-calls:latest
```

[hub]: https://hub.docker.com/r/jeremija/peer-calls

## From Git Source

```bash
git clone https://github.com/jeremija/peer-calls.git
cd peer-calls
npm install

# for production
npm run build
npm start

# for development
npm run start:watch
```

## Building Docker Image

```bash
git clone https://github.com/jeremija/peer-calls
cd peer-calls
docker build -t peer-calls .
docker run --rm -it -p 3000:3000 peer-calls:latest
```

# Configuration

There has been a breaking change in `v3.0.0`. The default binary provided via
NPM is now called peer-calls, while it used to be peercalls. This has been made
to make `npx peer-calls` work.

Version 3 also changed the way configuration works. Previously, `config` module
was used to read config files. To make things simpler, a default STUN
configuration will now be used by default. Config files can still be provided
via the `config/` folder in the working directory, but the extension read will
be `.yaml` instead of `.json`.

The config files are read in the following order:

- `node_modules/peer-calls/config/default.yaml`
- `node_modules/peer-calls/config/${NODE_ENV}.yaml`, if `NODE_ENV` is set
- `node_modules/peer-calls/config/local.yaml`
- `./config/default.yaml`
- `./config/${NODE_ENV}.yaml`, if `NODE_ENV` is set
- `./config/local.yaml`

No errors will be thrown if a file is not found, but an error will be thrown
when the required properties are not found. To debug configuration issues,
set the `DEBUG` environment variable to `DEBUG=peercalls,peercalls:config`.

Additionally, version 3 provides easier configuration via environment
variables. For example:

 - Set STUN/TURN servers: `PEERCALLS__ICE_SERVERS='[{"url": "stun:stun.l.google.com:19302", "urls": "stun:stun.l.google.com:19302"}]'`
 - Change base URL: `PEERCALLS__BASE_URL=/test` - app will now be accessible at `localhost:3000/test`
 - Enable HTTPS: `PEERCALLS__SSL='{"cert": "/path/to/cert.pem", "key": "/path/to/cert.key"}'`
 - Disable HTTPS: `PEERCALLS__SSL=undefined`
 - Listen on a different port: `PORT=3001` (default is `3000`)
 - Bind to specific IP or hostname: `BIND=127.0.0.1`

See [config/default.yaml][config] for sample configuration.

[config]: ./config/default.yaml

By default, the server will start on port `3000`. This can be modified by
setting the `PORT` environment variable to another number, or to a path for a
unix domain socket.

To access the server, go to http://localhost:3000 (or another port).

# Logging

By default, Peer Calls server will log only basic information. Client-side
logging is disabled by default.

Server-side logs can be configured via the `DEBUG` environment variable. Setting
it to `peercalls,peercalls:*` will enable all server-side logging:

- `DEBUG=peercalls,peercalls:* npm run start:server`

Client-side logs can be configured via `localStorage.DEBUG` and
`localStorage.LOG` variables:

- Setting `localStorage.LOG=1` enables logging of Redux actions and state
  changes
- Setting `localStorage.DEBUG=peercalls,peercalls:*` enables all other
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
13 on Mac OS.

Does not work on iOS 10, but should work on iOS 11 - would appreciate feedback!

For more details, see here:

- http://caniuse.com/#feat=rtcpeerconnection
- http://caniuse.com/#search=getUserMedia

# TODO

- [x] Do not require config files and allow configuration via environment
  variables. (Fixed in 23fabb0)
- [x] Show menu dialog before connecting (Fixed in 0b4aa45)
- [x] Reduce production build size by removing Pug. (Fixed in 2d14e5f c743f19)
- [x] Add ability to share files (Fixed in 3877893)
- [ ] Enable node cluster support (to scale vertically).
- [ ] Add Socket.IO support for Redis (to scale horizontally).
- [ ] Generate public keys for each client, and allow each client to accept,
      deny, and remember allowed/denied connections to specific peers.
- [ ] Add support for browser push notifications

# Contributing

See [Contributing](CONTRIBUTING.md) section.

If you encounter a bug, please open a new issue! Thank you ❤️

# License

[MIT](LICENSE)
