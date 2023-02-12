import React  from 'react'
import { ME } from '../constants'
import { ReceiverStatsParams } from '../reducers/receivers'
import { StreamWithURL } from '../reducers/streams'

export interface StatsProps {
  stream?: StreamWithURL
  peerId: string
  getReceiverStats: (
    params: ReceiverStatsParams,
  ) => Promise<RTCStatsReport | undefined>
  getSenderStats: (
    track: MediaStreamTrack,
  ) => Promise<{peerId: string, stats: RTCStatsReport}[]>
}

export interface StatsState {
  report: string
}

export default class Stats
extends React.PureComponent<StatsProps, StatsState> {
  timeout: NodeJS.Timeout | undefined
  state = {
    report: '',
  }

  componentDidMount() {
    this.refresh()
    this.timeout = setInterval(this.refresh, 1000)
  }

  refresh = async () => {
    let report = ''
    if (this.props.peerId === ME) {
      report = await this.fetchSenderStats()
    } else {
      report = await this.fetchReceiverStats()
    }

    this.setState({
      report,
    })
  }

  buildStatsReport = (stats: RTCStatsReport, sections: string[]): string => {
    let r = ''

    const set = new Set(sections)

    stats.forEach(v => {
      if (!set.has(v.type)) {
        return
      }

      const i = v as RTCInboundRTPStreamStats
      const o = v as RTCOutboundRTPStreamStats

      switch (v.type) {
      case 'codec':
        if (v.channels !== undefined) {
          r += 'Channels: ' + v.channels + '\n'
        }
        r += 'Clock rate: ' + v.clockRate + '\n'
        r += 'MIME Type: ' + v.mimeType + '\n'
        r += 'Payload Type: ' + v.payloadType + '\n'
        r += 'SDP FMTP Line: ' + v.sdpFmtpLine + '\n'
        break
      case 'inbound-rtp':
        r += 'SSRC: ' + i.ssrc + '\n'
        r += 'Bytes received: ' + i.bytesReceived + '\n'
        r += 'Packets received: ' + i.packetsReceived + '\n'
        r += 'Packets discarded: ' + v.packetsDiscarded + '\n'
        r += 'Packets lost: ' + i.packetsLost + '\n'
        if (i.firCount !== undefined) {
          r += 'FIR count: ' + i.firCount + '\n'
        }

        if (i.pliCount !== undefined) {
          r += 'PLI count: ' + i.pliCount + '\n'
        }

        if (i.nackCount !== undefined) {
          r += 'NACK count: ' + i.nackCount + '\n'
        }

        if (i.sliCount !== undefined) {
          r += 'SLI count: ' + i.sliCount + '\n'
        }
        break
      case 'outbound-rtp':
        r += 'SSRC: ' + o.ssrc + '\n'
        r += 'Bytes sent: ' + o.bytesSent + '\n'
        r += 'Packets sent: ' + o.packetsSent + '\n'

        if (o.firCount !== undefined) {
          r += 'FIR count: ' + o.firCount + '\n'
        }

        if (o.pliCount !== undefined) {
          r += 'PLI count: ' + o.pliCount + '\n'
        }

        if (o.nackCount !== undefined) {
          r += 'NACK count: ' + o.nackCount + '\n'
        }

        if (o.sliCount !== undefined) {
          r += 'SLI count: ' + o.sliCount + '\n'
        }

        if (o.roundTripTime !== undefined) {
          r += 'Round trip time: ' + o.roundTripTime + '\n'
        }
      break
      default:
          // Do nothing.
      }
    })

    return r
  }
  fetchReceiverStats = async () => {
    const { stream, getReceiverStats } = this.props

    if (!stream) {
      return 'No stream'
    }

    const streamId = stream.streamId

    // Keep video/audio order consistent.
    const tracks = [
      ...stream.stream.getVideoTracks(),
      ...stream.stream.getAudioTracks(),
    ]

    const tps = tracks.map(track => {
      return {
        track,
        promise: getReceiverStats({
          streamId,
          trackId: track.id,
        }),
      }
    })

    const reports: string[] = []
    const sections = ['codec', 'inbound-rtp']

    for (const tp of tps) {
      const { track, promise } = tp

      let r = ''

      r += `${track.kind.toUpperCase()} ${track.id}\n`

      try {
        const stats = await promise
        if (stats) {
          r += this.buildStatsReport(stats, sections)
        } else {
          r += 'No report available\n'
        }
      } catch (err) {
        r += 'Error ' + err + '\n'
      }

      reports.push(r)
    }

    return reports.join('\n')
  }
  fetchSenderStats = async () => {
    const { stream, getSenderStats } = this.props

    if (!stream) {
      return 'No stream'
    }

    // Keep video/audio order consistent.
    const tracks = [
      ...stream.stream.getVideoTracks(),
      ...stream.stream.getAudioTracks(),
    ]

    const tps = tracks.map(track => {
      return {
        track,
        promise: getSenderStats(track),
      }
    })

    const reports: string[] = []
    const sections = ['codec', 'outbound-rtp']

    for (const tp of tps) {
      const { track, promise } = tp

      let r = `${track.kind.toUpperCase()} ${track.id}\n`
      let statsPerPeer = []

      try {
        statsPerPeer = await promise
      } catch (err) {
        r += 'Error ' + err + '\n'
        continue
      }

      if (!statsPerPeer.length) {
        r += 'No report available\n'
        continue
      }

      statsPerPeer.forEach(s => {
        const { peerId, stats } = s

        r += `Peer ID: ${peerId}` + '\n'
        r += this.buildStatsReport(stats, sections)
      })

      reports.push(r)
    }

    return reports.join('\n')
  }

  componentWillUnmount() {
    if (this.timeout) {
      clearTimeout(this.timeout)
    }
  }

  render() {
    return this.state.report
  }
}
