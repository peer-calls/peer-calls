#!/usr/bin/env node
'use strict'
const router = require('express').Router()

router.get('/', (req, res) => {
  res.render('index')
})

module.exports = router
