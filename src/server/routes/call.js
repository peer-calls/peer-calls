#!/usr/bin/env node
'use strict';
const express = require('express');
const router = express.Router();
const uuid = require('uuid');

router.use((req, res, next) => {
  next();
});

router.get('/', (req, res) => {
	let prefix = 'call/';
	if (req.url.charAt(req.url.length - 1) === '/') prefix = '';
	res.redirect(prefix + uuid.v4());
});

router.get('/:callId', (req, res) => {
  res.render('call', {
    callId: encodeURIComponent(req.params.callId)
  });
});

module.exports = router;
