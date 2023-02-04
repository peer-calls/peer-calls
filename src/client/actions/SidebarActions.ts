import { SIDEBAR_HIDE, SIDEBAR_SHOW, SIDEBAR_TOGGLE } from '../constants'

export type Panel = 'chat' | 'users' | 'settings'

export const sidebarPanelChat: Panel = 'chat'
export const sidebarPanelSettings: Panel = 'settings'
export const sidebarPanelUsers: Panel = 'users'

export interface SidebarToggleAction {
  type: 'SIDEBAR_TOGGLE'
}

export interface SidebarHideAction {
  type: 'SIDEBAR_HIDE'
}

export interface SidebarShowAction {
  type: 'SIDEBAR_SHOW'
  payload: SidebarShowPayload
}

export interface SidebarShowPayload {
  panel: Panel
}

export function sidebarToggle(): SidebarToggleAction {
  return {
    type: SIDEBAR_TOGGLE,
  }
}

export function sidebarShow(panel?: Panel): SidebarShowAction {
  panel = panel || sidebarPanelChat

  return {
    type: SIDEBAR_SHOW,
    payload: {
      panel,
    },
  }
}

export function sidebarHide(): SidebarHideAction {
  return {
    type: SIDEBAR_HIDE,
  }
}

export type SidebarAction =
  SidebarToggleAction |
  SidebarHideAction |
  SidebarShowAction
