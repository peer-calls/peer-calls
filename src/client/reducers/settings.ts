import { GridKind, GridKinds, SettingsAction } from '../actions/SettingsActions'
import { SETTINGS_GRID_AUTO, SETTINGS_GRID_SET, SETTINGS_SHOW_MINIMIZED_TOOLBAR_TOGGLE } from '../constants'
import { localStorage } from '../window'

export interface SettingsState {
  showMinimizedToolbar: boolean
  gridKind: GridKind
}

const settingsKey = 'settings'

function init(): SettingsState {
  return {
    showMinimizedToolbar: true,
    gridKind: SETTINGS_GRID_AUTO,
  }
}

function withDefault(
  state: Partial<SettingsState> | null, init: SettingsState,
): SettingsState {

  if (!state) {
    return init
  }

  return {
    showMinimizedToolbar:
      typeof state.showMinimizedToolbar === 'boolean'
        ? state.showMinimizedToolbar
        : init.showMinimizedToolbar,
    gridKind : GridKinds[state.gridKind as string] || SETTINGS_GRID_AUTO,
  }
}

function load(): SettingsState {
  const def = init()

  let loaded: Partial<SettingsState> | null = null

  try {
    loaded = JSON.parse(localStorage.getItem(settingsKey)!)
  } catch {
    // Do nothing
  }

  if (!loaded) {
    return def
  }

  return withDefault(loaded, def)
}

function save(state: SettingsState): SettingsState {
  try {
    localStorage.setItem(settingsKey, JSON.stringify(state))
  } catch {
    // Do nothing.
  }

  return state
}

export default function settings(
  state: SettingsState = load(),
    action: SettingsAction,
): SettingsState {
  switch (action.type) {
  case SETTINGS_SHOW_MINIMIZED_TOOLBAR_TOGGLE:
    return save({
      ...state,
      showMinimizedToolbar: !state.showMinimizedToolbar,
    })
  case SETTINGS_GRID_SET:
    return save({
      ...state,
      gridKind: action.payload.gridKind,
    })
  default:
    return state
  }
}
