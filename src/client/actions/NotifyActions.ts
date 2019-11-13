import * as constants from '../constants'
import { Dispatch  } from 'redux'
import _ from 'underscore'
import { ThunkResult } from '../store'

const TIMEOUT = 5000

function format (string: string, args: string[]) {
  string = args
  .reduce((string, arg, i) => string.replace('{' + i + '}', arg), string)
  return string
}

export type NotifyType = 'info' | 'warning' | 'error'

function notify(dispatch: Dispatch, type: NotifyType, args: string[]) {
  const string = args[0] || ''
  const message = format(string, Array.prototype.slice.call(args, 1))
  const id = _.uniqueId('notification')
  const payload: Notification = { id, type, message }

  setTimeout(() => {
    dispatch(dismissNotification(id))
  }, TIMEOUT)

  return addNotification(payload)
}

export const info = (...args: any[]): ThunkResult<NotificationAddAction> => {
  return dispatch => notify(dispatch, 'info', args)
}

export const warning = (...args: any[]): ThunkResult<NotificationAddAction> => {
  return dispatch => notify(dispatch, 'warning', args)
}

export const error = (...args: any[]): ThunkResult<NotificationAddAction> => {
  return dispatch => notify(dispatch, 'error', args)
}

function addNotification(payload: Notification): NotificationAddAction {
  return {
    type: constants.NOTIFY,
    payload,
  }
}

function dismissNotification(id: string): NotificationDismissAction {
  return {
    type: constants.NOTIFY_DISMISS,
    payload: { id },
  }
}

export interface Notification {
  id: string
  type: NotifyType
  message: string
}

export interface NotificationAddAction {
  type: 'NOTIFY'
  payload: Notification
}

export interface NotificationDismissAction {
  type: 'NOTIFY_DISMISS'
  payload: { id: string }
}

export interface NotificationClearAction {
  type: 'NOTIFY_CLEAR'
}

export const clear = (): NotificationClearAction => ({
  type: constants.NOTIFY_CLEAR,
})

export interface Alert {
  action?: string
  dismissable: boolean
  message: string
  type: NotifyType
}

export interface AlertAddAction {
  type: 'ALERT'
  payload: Alert
}

export function alert (message: string, dismissable = false): AlertAddAction {
  return {
    type: constants.ALERT,
    payload: {
      action: dismissable ? 'Dismiss' : '',
      dismissable: !!dismissable,
      message,
      type: 'warning',
    },
  }
}

export interface AlertDismissAction {
  type: 'ALERT_DISMISS'
  payload: Alert
}

export const dismissAlert = (alert: Alert): AlertDismissAction => {
  return {
    type: constants.ALERT_DISMISS,
    payload: alert,
  }
}

export interface AlertClearAction {
  type: 'ALERT_CLEAR'
}

export const clearAlerts = (): AlertClearAction => {
  return {
    type: constants.ALERT_CLEAR,
  }
}

export type AlertActionType =
  AlertAddAction |
  AlertDismissAction |
  AlertClearAction

export type NotificationActionType =
  NotificationAddAction |
  NotificationDismissAction |
  NotificationClearAction
