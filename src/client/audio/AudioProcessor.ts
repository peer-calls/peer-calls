import _debug from 'debug'
import { AudioMessage, VUMeter } from './types'
import { AudioContext, AudioWorkletNode, config, MediaStream } from '../window'

const debug = _debug('peercalls')

// AudioPipeline holds references to the parts of the audio processing pipeline
// related to a certain audio track.
interface AudioPipeline {
  track: MediaStreamTrack
  source: MediaStreamTrackAudioSourceNode
  node: AudioWorkletNode
  removeListener: () => void
}

// Context wraps AudioContext and initiates volume processing for each added
// track.
export class AudioProcessor {
  ctx?: AudioContext
  tracks: Record<string, AudioPipeline> = {}
  subs: Record<string, Record<string, AudioMessageCallback>> = {}
  subCount = 0

  async init() {
    try {
      await this.unsafeInit()
    } catch (err) {
      debug('Environment does not support AudioWorklets: %s', err)
    }
  }

  async unsafeInit() {
    if (this.ctx) {
      return
    }

    this.ctx = new AudioContext()

    const workletURL = config.baseUrl + '/static/audio.worklet.js'
    await this.ctx.audioWorklet.addModule(workletURL)
  }

  // addTrack adds a MediaStreamTrack and initiates audio processing.
  addTrack(streamId: string, track: MediaStreamTrack) {
    try {
      this.unsafeAddTrack(streamId, track)
    } catch (err) {
      debug('AudioProcessor.addTrack failed: %s', err)
    }

  }

  private unsafeAddTrack(streamId: string, track: MediaStreamTrack) {
    if (!this.ctx) {
      debug('AudioProcessor.unsafeAddTrack, ctx not initialized: %s', streamId)
      return
    }

    if (track.kind !== 'audio') {
      debug('AudioProcessor.unsafeAddTrack, track is not audio: %s', streamId)
      return
    }

    const stream = new MediaStream()
    stream.addTrack(track)

    const source = this.ctx.createMediaStreamSource(stream)

    const node = new AudioWorkletNode(this.ctx, VUMeter)

    source.connect(node)

    debug('AudioProcessor.addTrack: %s', streamId)

    const sub = (event: MessageEvent<AudioMessage>) => {
      const subs = this.subs[streamId] || {}

      Object.keys(subs).forEach(key => {
        subs[key](event.data)
      })
    }

    node.port.onmessage = sub

    const removeListener = () => {
      node.port.onmessage = null
    }

    // // Connect the node to contrxt output. I don't think this is needed.
    // node.connect(this.ctx.destination)

    this.tracks[streamId] = {
      track,
      source,
      node,
      removeListener,
    }
  }

  replaceTrack(streamId: string, track: MediaStreamTrack) {
    if (track.kind !== 'audio') {
      return
    }

    this.removeTrack(streamId)
    this.addTrack(streamId, track)
  }

  // removeTrack disconnects the source from the audio worklet node.
  removeTrack(streamId: string) {
    const pipeline = this.tracks[streamId]
    if (!pipeline) {
      return
    }

    debug('AudioProcessor.removeTrack: %s', streamId)

    const { source, node, removeListener } = this.tracks[streamId]

    removeListener()

    source.disconnect(node)

    delete this.tracks[streamId]
  }

  // subscribe creates a subscription to audio updates and returns an function
  // which can be used to unsubscribe.
  subscribe(streamId: string, callback: AudioMessageCallback): Unsubscribe {
    debug('AudioProcessor.subscribe: %s', streamId)

    const subs = this.subs[streamId] = this.subs[streamId] || {}

    const subId = ++this.subCount

    subs[subId] = callback

    return () => {
      delete subs[subId]
      if (Object.keys(subs).length === 0) {
        delete this.subs[streamId]
      }
    }
  }
}

export const audioProcessor = new AudioProcessor()

type AudioMessageCallback = (msg: AudioMessage) => void
type Unsubscribe = () => void
