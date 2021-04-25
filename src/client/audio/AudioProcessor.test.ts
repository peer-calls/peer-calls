jest.mock('../window')

import { MediaStreamTrack } from '../window'
import { AudioProcessor } from './index'

describe('audio/AudioProcessor', () => {

  describe('addTrack', () => {
    let p: AudioProcessor
    beforeEach(async () => {
      p = new AudioProcessor()
      await p.unsafeInit()
    })

    it('creates audio processing pipeline for track', () => {
      const track = new MediaStreamTrack()
      ;(track as any).kind = 'audio'
      p.addTrack('s1', track)

      const dispatch = p.tracks['s1'].node.port.onmessage as any
      const subFn = jest.fn()
      const unsubscribe = p.subscribe('s1', subFn)

      const event = {
        data: {
         type: 'volume',
         volume: 0.9 ,
        },
      }

      dispatch(event as any)
      expect(subFn.mock.calls).toEqual([[ event.data ]])

      subFn.mockClear()
      unsubscribe()

      dispatch(event as any)
      expect(subFn.mock.calls).toEqual([])

      p.removeTrack('s1')
    })
  })

})
