#!/usr/bin/env node
'use strict';
const config = require('config');
const turn = require('../turn.js');
const router = require('express').Router();
const uuid = require('uuid');

const cfgIceServers = config.get('iceServers');

router.get('/', (req, res) => {
  let prefix = 'call/';
  if (req.originalUrl.charAt(req.originalUrl.length - 1) === '/') prefix = '';
  res.redirect(prefix + uuid.v4());
});

router.get('/:callId', (req, res) => {
  const iceServers = turn.processServers(cfgIceServers);
  res.render('call', {
    callId: encodeURIComponent(req.params.callId),
    iceServers
  });
});

module.exports = router;
