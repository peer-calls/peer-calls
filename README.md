# Peer Calls

[![Build Status](https://travis-ci.org/jeremija/peer-calls.svg?branch=master)](https://travis-ci.org/jeremija/peer-calls)

WebRTC peer to peer calls for everyone. See it live in action at
[peercalls.com](https://peercalls.com).

Work in progress.

# Install & Run
REQUIRES Node.js v5.10.1 [https://nodejs.org/en/](https://nodejs.org/en/)

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

On your other machine or mobile device open the url:

```bash
http://<your_ip_or_localhost>:3000
```
(Note: On Android you may have to select a notification on the pulldown menu to connect if you are using Chrome)

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
