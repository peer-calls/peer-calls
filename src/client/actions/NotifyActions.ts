import uniqueId from 'lodash/uniqueId'
import * as constants from '../constants'

function format (string: string, args: string[]) {
  string = args
  .reduce((string, arg, i) => string.replace('{' + i + '}', arg), string)
  return string
}

export type NotifyType = 'info' | 'warning' | 'error'

function notify(type: NotifyType, args: string[]) {
  const string = args[0] || ''
  const message = format(string, Array.prototype.slice.call(args, 1))
  const id = uniqueId('notification')
  const payload: Notification = { id, type, message }

  return addNotification(payload)
}

export const info = (...args: any[]): NotificationAddAction => {
  return notify('info', args)
}

export const warning = (...args: any[]): NotificationAddAction => {
  return notify('warning', args)
}

export const error = (...args: any[]): NotificationAddAction => {
  return notify('error', args)
}

function addNotification(payload: Notification): NotificationAddAction {
  return {
    type: constants.NOTIFY,
    payload,
  }
}

export function dismissNotification(id: string): NotificationDismissAction {
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

export type NotificationActionType =
  NotificationAddAction |
  NotificationDismissAction |
  NotificationClearAction
