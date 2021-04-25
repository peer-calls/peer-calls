import { VolumeMessage, VUMeter } from './types'

declare const sampleRate: number
declare const registerProcessor: (
    name: string,
    processorCtor: (new (
      options?: AudioWorkletNodeOptions
    ) => AudioWorkletProcessor) & {
      parameterDescriptors?: AudioParamDescriptor[]
    }
  ) => void

interface AudioWorkletProcessor {
  readonly port: MessagePort
  process(
    inputs: Float32Array[][],
    outputs: Float32Array[][],
    parameters: Record<string, Float32Array>
  ): boolean
}

declare const AudioWorkletProcessor: {
  prototype: AudioWorkletProcessor
  new (options?: AudioWorkletNodeOptions): AudioWorkletProcessor
}

export default function init() {
  class VolumeProcessor extends AudioWorkletProcessor {
    private volume = 0
    private updateIntervalMillis = 25
    private nextUpdateFrame = this.updateIntervalMillis

    private intervalInFrames() {
      return this.updateIntervalMillis * sampleRate / 1000
    }

    process(
      inputs: Float32Array[][],
      // outputs: Float32Array[][],
      // parameters: Record<string, Float32Array>,
    ): boolean {
      if (inputs.length === 0) {
        return true
      }

      const input = inputs[0]

      if (input.length === 0) {
        return true
      }

      const samples = input[0]

      let sum = 0
      let rms = 0

      for (let i = 0; i < samples.length; i++) {
        sum += samples[i] * samples[i]
      }

      rms = Math.sqrt(sum / samples.length)

      this.volume = Math.max(rms, this.volume * 0.99)

      this.nextUpdateFrame -= samples.length
      if (this.nextUpdateFrame < 0) {
        const message: VolumeMessage = {
          type: 'volume',
          volume: this.volume,
        }

        this.nextUpdateFrame += this.intervalInFrames()
        this.port.postMessage(message)
      }

      return true
    }
  }

  registerProcessor(VUMeter, VolumeProcessor)
  console.log('VUMeter initiated')
}

init()
