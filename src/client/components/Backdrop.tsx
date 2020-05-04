import React from 'react'
import ReactDOM from 'react-dom'

export interface BackdropProps {
  visible: boolean
  onClick: () => void
}

export class Backdrop extends React.PureComponent<BackdropProps> {
  render() {
    if (!this.props.visible) {
      return null
    }

    return ReactDOM.createPortal(
      <section className='dropdown-backdrop' onClick={this.props.onClick} />,
      document.body,
    )
  }
}

