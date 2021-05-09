import { SettingsAction } from '../actions/SettingsActions'
import { SETTINGS_SHOW_MINIMIZED_TOOLBAR_TOGGLE } from '../constants'
import { localStorage } from '../window'

export interface SettingsState {
  showMinimizedToolbar: boolean
}

const settingsKey = 'settings'

function init(): SettingsState {
  return {
    showMinimizedToolbar: true,
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
      typeof state.showMinimizedToolbar === 'boolean' ?
      state.showMinimizedToolbar : init.showMinimizedToolbar,
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
  default:
    return state
  }
}
