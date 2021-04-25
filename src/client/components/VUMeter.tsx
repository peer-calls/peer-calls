import React from 'react'
import { AudioMessage, audioProcessor } from '../audio'

export interface VUMeterProps {
  streamId?: string
}

export interface VUMeterState {
  volume: number
}

function noop() {}

export default class VUMeter
extends React.PureComponent<VUMeterProps, VUMeterState> {
  private unsubscribe: () => void = noop

  state: VUMeterState = {
    volume: 0,
  }

  componentDidMount() {
    this.resubscribe()
  }

  componentDidUpdate(prevProps: VUMeterProps) {
    if (prevProps.streamId !== this.props.streamId) {
      this.resubscribe()
    }
  }

  componentWillUnmount() {
    this.unsubscribe()
  }

  resubscribe() {
    this.unsubscribe()

    if (!this.props.streamId) {
      return
    }

    const unsubscribe = audioProcessor
    .subscribe(this.props.streamId, this.handleAudioMessage)

    this.unsubscribe = () => {
      unsubscribe()
      this.unsubscribe = noop
    }
  }

  handleAudioMessage = (msg: AudioMessage) => {
    if (msg.type !== 'volume') {
      return
    }

    // ease out function.
    const scaled = 1 - Math.pow(1 - msg.volume, 3)

    this.setState({
      volume: Math.round(scaled * 5),
    })
  }

  render() {
    const className = 'vu-meter vu-meter-level-' + this.state.volume

    return (
      <div className={className}>
      </div>
    )
  }
}
