window.localStorage = { debug: true }
import logger from 'redux-logger'
const store = require('../store.js')

describe('store', () => {

  it('should load logger middleware', () => {
    expect(store.middlewares.some(m => m === logger)).toBeTruthy()
  })

})
