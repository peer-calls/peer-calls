export type AudioMessage = VolumeMessage

export const VUMeter = 'vu-meter'

export interface VolumeMessage {
  type: 'volume'
  volume: number
}
