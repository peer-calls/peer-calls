import configureStore from 'redux-mock-store'
import { middlewares } from '../middlewares.js'
export default configureStore(middlewares)({})
