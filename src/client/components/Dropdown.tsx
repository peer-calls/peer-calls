import React from 'react'
import classnames from 'classnames'

export interface DropdownProps {
  label: string | React.ReactElement
}

export interface DropdownState {
  open: boolean
}

export class Dropdown
extends React.PureComponent<DropdownProps, DropdownState> {
  state = { open: false }

  handleClick = () => {
    this.setState({ open: !this.state.open })
  }
  render() {
    const classNames = classnames('dropdown-list', {
      'dropdown-list-open': this.state.open,
    })

    return (
      <div className='dropdown'>
        <button onClick={this.handleClick} >{this.props.label}</button>
        <ul className={classNames}>
          {this.props.children}
        </ul>
      </div>
    )
  }
}
