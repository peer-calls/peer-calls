import logger from 'redux-logger'
import reducer from './reducers'
import thunk from 'redux-thunk'
import { applyMiddleware, createStore } from 'redux'

export default createStore(
  reducer,
  applyMiddleware(thunk, logger)
)
