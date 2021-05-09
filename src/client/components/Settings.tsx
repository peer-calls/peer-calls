import uniqueId from 'lodash/uniqueId'
import React from 'react'
import { connect } from 'react-redux'
import { showMinimizedToolbarToggle } from '../actions/SettingsActions'
import { SettingsState } from '../reducers/settings'
import { State } from '../store'


export interface SettingsProps extends SettingsState {
  showMinimizedToolbarToggle: typeof showMinimizedToolbarToggle
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
      <li>
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
      </li>
    )
  }
}

class Settings extends React.PureComponent<SettingsProps> {
  render() {
    const {
      showMinimizedToolbar,
      showMinimizedToolbarToggle,
    } = this.props

    return (
      <div className='settings'>
        <ul className='settings-list'>
          <Checkbox
            className='settings-show-minimized-toolbar-toggle'
            checked={showMinimizedToolbar}
            onChange={showMinimizedToolbarToggle}
            label='Show Minimized Toolbar'
          />
        </ul>
        <div></div> {/*necessary for flex to stretch */}
      </div>
    )
  }
}

const bind = {
  showMinimizedToolbarToggle,
}

function mapStateToProps(state: State) {
  return {
    ...state.settings,
  }
}

export default connect(mapStateToProps, bind)(Settings)
