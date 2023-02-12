import uniqueId from 'lodash/uniqueId'
import React from 'react'
import { connect } from 'react-redux'
import { showMinimizedToolbarToggle, setGridKind, showAllStatsToggle } from '../actions/SettingsActions'
import { SETTINGS_GRID_ASPECT, SETTINGS_GRID_AUTO, SETTINGS_GRID_LEGACY } from '../constants'
import { SettingsState } from '../reducers/settings'
import { State } from '../store'

export interface SettingsProps extends SettingsState {
  showMinimizedToolbarToggle: typeof showMinimizedToolbarToggle
  setGridKind: typeof setGridKind
  showAllStatsToggle: typeof showAllStatsToggle
}

interface CheckboxProps {
  label: string
  className: string
  onChange: () => void
  checked: boolean
}

class Checkbox extends React.PureComponent<CheckboxProps> {
  uniqueId: string
  constructor(props: CheckboxProps) {
    super(props)
    this.uniqueId = uniqueId('checkbox-')
  }
  handleChange = () => {
    this.props.onChange()
  }
  render() {
    return (
      <label htmlFor={this.uniqueId}>
        <input
          id={this.uniqueId}
          className={this.props.className}
          type='checkbox'
          checked={this.props.checked}
          onChange={this.handleChange}
        />
        {this.props.label}
      </label>
    )
  }
}

interface RadioProps<Value> {
  label: string
  name: string
  className: string
  onChange: (value: Value) => void
  value: Value
  currentValue: Value
}

class Radio<Value extends string>
extends React.PureComponent<RadioProps<Value>> {
  uniqueId: string
  constructor(props: RadioProps<Value>) {
    super(props)
    this.uniqueId = uniqueId('radio-')
  }
  handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value as Value
    this.props.onChange(value)
  }
  render() {
    return (
      <label htmlFor={this.uniqueId}>
        <input
          id={this.uniqueId}
          className={this.props.className}
          name={this.props.name}
          type='radio'
          value={this.props.value as string}
          onChange={this.handleChange}
          checked={this.props.value == this.props.currentValue}
        />
        {this.props.label}
      </label>
    )
  }
}

class Settings extends React.PureComponent<SettingsProps> {
  render() {
    const {
      showMinimizedToolbar,
      showMinimizedToolbarToggle,
      gridKind,
      setGridKind,
      showAllStats,
      showAllStatsToggle,
    } = this.props

    return (
      <div className='settings'>
        <ul className='settings-list'>
          <li>
            <Checkbox
              className='settings-show-minimized-toolbar-toggle'
              checked={showMinimizedToolbar}
              onChange={showMinimizedToolbarToggle}
              label='Show Minimized Toolbar'
            />
          </li>
          <li>
            <h3 className='settings-title'>Video grid</h3>
            <Radio
              className='settings-grid-kind settings-grid-kind-auto'
              onChange={setGridKind}
              name='settings-grid-kind'
              value={SETTINGS_GRID_AUTO}
              currentValue={gridKind}
              label='Auto'
            />
            <Radio
              className='settings-grid-kind settings-grid-kind-legacy'
              onChange={setGridKind}
              name='settings-grid-kind'
              value={SETTINGS_GRID_LEGACY}
              currentValue={gridKind}
              label='Legacy fill'
            />
            <Radio
              className='settings-grid-kind settings-grid-kind-aspect'
              onChange={setGridKind}
              name='settings-grid-kind'
              value={SETTINGS_GRID_ASPECT}
              currentValue={gridKind}
              label='Aspect ratio'
            />
            <p className='settings-description'>
              {gridKind === SETTINGS_GRID_AUTO && (
                <span>
                  Automatically switch between legacy grid and preserving the
                  aspect ratio when there are more than two participants
                </span>
              )}
              {gridKind === SETTINGS_GRID_LEGACY && (
                <span>
                  Always try to fill in as much screen space with videos as
                  possible, even if it means cropping the picture
                </span>
              )}
              {gridKind === SETTINGS_GRID_ASPECT && (
                <span>
                  Try to preserve the video aspect ratio, even if it means
                  leaving empty space on the sides.
                </span>
              )}
            </p>
          </li>
          <li>
            <Checkbox
              className='settings-show-all-stats-toggle'
              checked={showAllStats}
              onChange={showAllStatsToggle}
              label='Show all WebRTC stats'
            />
          </li>
        </ul>
        <div></div> {/*necessary for flex to stretch */}
      </div>
    )
  }
}

const bind = {
  showMinimizedToolbarToggle,
  setGridKind,
  showAllStatsToggle,
}

function mapStateToProps(state: State) {
  return {
    ...state.settings,
  }
}

export default connect(mapStateToProps, bind)(Settings)
