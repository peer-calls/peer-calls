# Peer Calls

[![Build Status](https://travis-ci.org/jeremija/peer-calls.svg?branch=master)](https://travis-ci.org/jeremija/peer-calls)

WebRTC peer to peer calls for everyone. See it live in action at
[peercalls.com](https://peercalls.com).

Work in progress.

# Requirements
 - Node.js 5 [https://nodejs.org/en/](https://nodejs.org/en/)

# Installation & Running

From git source:

```bash
git clone https://github.com/jeremija/peer-calls.git
cd peer-calls
npm install
npm run build
npm start
```

If you successfully completed the above steps, your commandline/terminal should
show that your node server is listening.

On your other machine or mobile device open the url:

```bash
http://<your_ip_or_localhost>:3000

# Testing

```bash
npm install
npm test
```

# Browser Support

Tested on Firefox and Chrome, including mobile versions.

Does not work on iOS 10, but should work on iOS 11 (untested).

For more details, see here:

- http://caniuse.com/#feat=rtcpeerconnection
- http://caniuse.com/#search=getUserMedia

# Contributing

See [Contributing](CONTRIBUTING.md) section.

# License

MIT
