import { SETTINGS_SHOW_MINIMIZED_TOOLBAR_TOGGLE } from '../constants'

export interface ShowMinimizedToolbarToggleAction {
  type: 'SETTINGS_SHOW_MINIMIZED_TOOLBAR_TOGGLE'
}

export function showMinimizedToolbarToggle(
): ShowMinimizedToolbarToggleAction {
  return {
    type: SETTINGS_SHOW_MINIMIZED_TOOLBAR_TOGGLE,
  }
}

export type SettingsAction =
  ShowMinimizedToolbarToggleAction
