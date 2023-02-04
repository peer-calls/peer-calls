import { Panel, SidebarAction, SidebarShowPayload } from '../actions/SidebarActions'
import { SIDEBAR_HIDE, SIDEBAR_SHOW, SIDEBAR_TOGGLE } from '../constants'

export interface SidebarState {
  panel: Panel
  visible: boolean
}

const defaultState: SidebarState = {
  panel: 'chat',
  visible: false,
}

function toggle(state: SidebarState): SidebarState {
  return {
    ...state,
    visible: !state.visible,
  }
}

function show(
  state: SidebarState,
  payload: SidebarShowPayload,
): SidebarState {
  return {
    ...state,
    panel: payload.panel,
    visible: true,
  }
}

function hide(state: SidebarState): SidebarState {
  return {
    ...state,
    visible: false,
  }
}

export default function sidebar(
  state = defaultState,
  action: SidebarAction,
): SidebarState {
  switch (action.type) {
  case SIDEBAR_TOGGLE:
    return toggle(state)
  case SIDEBAR_SHOW:
    return show(state, action.payload)
  case SIDEBAR_HIDE:
    return hide(state)
  default:
    return state
  }
}
