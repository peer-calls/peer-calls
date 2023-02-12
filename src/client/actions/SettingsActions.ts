import { SETTINGS_SHOW_MINIMIZED_TOOLBAR_TOGGLE, SETTINGS_GRID_SET, SETTINGS_GRID_AUTO, SETTINGS_GRID_LEGACY, SETTINGS_GRID_ASPECT, SETTINGS_SHOW_ALL_STATS_TOGGLE } from '../constants'

export interface ShowMinimizedToolbarToggleAction {
  type: 'SETTINGS_SHOW_MINIMIZED_TOOLBAR_TOGGLE'
}

export interface UseFlexLayoutToggleAction {
  type: 'SETTINGS_GRID_SET'
  payload: {
    gridKind: GridKind
  }
}

export interface ShowAllStatsToggleAction {
  type: 'SETTINGS_SHOW_ALL_STATS_TOGGLE'
}

export function showMinimizedToolbarToggle(
): ShowMinimizedToolbarToggleAction {
  return {
    type: SETTINGS_SHOW_MINIMIZED_TOOLBAR_TOGGLE,
  }
}

export function showAllStatsToggle(
): ShowAllStatsToggleAction {
  return {
    type: SETTINGS_SHOW_ALL_STATS_TOGGLE,
  }
}

export type GridKind =
  'SETTINGS_GRID_AUTO' |
  'SETTINGS_GRID_LEGACY' |
  'SETTINGS_GRID_ASPECT'

export const GridKinds: Record<string, GridKind> = {
  SETTINGS_GRID_AUTO: SETTINGS_GRID_AUTO,
  SETTINGS_GRID_LEGACY: SETTINGS_GRID_LEGACY,
  SETTINGS_GRID_ASPECT: SETTINGS_GRID_ASPECT,
}

export function setGridKind(
  gridKind: GridKind,
): UseFlexLayoutToggleAction {
  return {
    type: SETTINGS_GRID_SET,
    payload: {
      gridKind,
    },
  }
}

export type SettingsAction =
  ShowMinimizedToolbarToggleAction |
  UseFlexLayoutToggleAction |
  ShowAllStatsToggleAction
