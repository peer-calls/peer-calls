# Remote Control Server

[![Build Status](https://travis-ci.org/jeremija/remote-control-server.svg?branch=master)](https://travis-ci.org/jeremija/remote-control-server)

Remote control your PC from your web browser on your other PC or mobile device.

Supports mouse movements, scrolling, clicking and keyboard input.

Work in progress.

<img src="http://i.imgur.com/38MzUIg.png" width="400px">
<img src="http://i.imgur.com/cn1IUK8.png" width="400px">
<img src="http://i.imgur.com/xtpgXoG.png" width="400px">

# Install & Run

Install from npm:

```bash
npm install -g remote-control-server
remote-control-server
```

or use from git source:

```bash
git clone https://github.com/jeremija/remote-control-server.git
cd node-mobile-remote
npm install
npm start
```

On your other machine or mobile device open the url:

```bash
http://192.168.0.10:3000
```

Replace `192.168.0.10` with the LAN IP address of your server.

# Note

This package requires [robotjs](https://www.npmjs.com/package/robotjs) so make
sure you have the required prerequisites installed for compiling that package.

# license

MIT
