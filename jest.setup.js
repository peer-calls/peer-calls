const { TextEncoder, TextDecoder } = require('util')

global.window.TextEncoder = TextEncoder
global.window.TextDecoder = TextDecoder
