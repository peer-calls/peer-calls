import logger from 'redux-logger'
import { create } from './middlewares'

describe('store', () => {

  it('should load logger middleware', () => {
    expect(create(true)).toContain(logger)
    expect(create(false)).not.toContain(logger)
  })

})
