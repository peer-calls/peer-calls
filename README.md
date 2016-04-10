# Peer Calls

[![Build Status](https://travis-ci.org/jeremija/peer-calls.svg?branch=master)](https://travis-ci.org/jeremija/peer-calls)

WebRTC peer to peer calls for everyone. See it live in action at
[peercalls.com](https://peercalls.com).

Work in progress.

# Install & Run

Note: You must have node version 5.1xx installed! 
If you accidentally entered npm install before upgrading to node version 5.1xx, simply delete the node module folder from your repository, upgrade to your node version, and repeat the npm install step.

From git source:

```bash
git clone https://github.com/jeremija/peer-calls.git
cd peer-calls
npm install
npm start
```

If you successfully completed the above steps, your commandline/terminal should show that your node server is listening.

![Alt text](http://imgur.com/wQ8RoVW "npm start")

On your other machine or mobile device open the url:

```bash
http://192.168.0.10:3000
```

# Running the tests

```bash
npm install
npm test
```

Replace `192.168.0.10` with the LAN IP address of your server.

# Contributing

See [Contributing](CONTRIBUTING.md) section.

# License

MIT
