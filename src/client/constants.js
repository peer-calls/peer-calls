import { PENDING, FULFILLED, REJECTED } from 'redux-promise-middleware'
export const ME = '_me_'

export const INIT = 'INIT'
export const INIT_PENDING = `${INIT}_${PENDING}`
export const INIT_FULFILLED = `${INIT}_${FULFILLED}`
export const INIT_REJECTED = `${INIT}_${REJECTED}`

export const ALERT = 'ALERT'
export const ALERT_DISMISS = 'ALERT_DISMISS'
export const ALERT_CLEAR = 'ALERT_CLEAR'

export const NOTIFY = 'NOTIFY'
export const NOTIFY_DISMISS = 'NOTIFY_DISMISS'
export const NOTIFY_CLEAR = 'NOTIFY_CLEAR'

export const STREAM_ADD = 'STREAM_ADD'
export const STREAM_ACTIVATE = 'STREAM_ACTIVATE'
export const STREAM_REMOVE = 'STREAM_REMOVE'
