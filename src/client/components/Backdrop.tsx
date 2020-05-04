import React from 'react'

export interface BackdropProps {
  visible: boolean
  onClick: () => void
}

export class Backdrop extends React.PureComponent<BackdropProps> {
  render() {
    if (!this.props.visible) {
      return null
    }

    return <section
      className='dropdown-backdrop'
      onClick={this.props.onClick}
    />
  }
}

