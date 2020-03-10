import { makeAction, AsyncAction } from '../async'
import { MEDIA_AUDIO_CONSTRAINT_SET, MEDIA_VIDEO_CONSTRAINT_SET, MEDIA_ENUMERATE, MEDIA_STREAM, ME, STREAM_TYPE_CAMERA, STREAM_TYPE_DESKTOP } from '../constants'
import _debug from 'debug'
import { AddStreamPayload } from './StreamActions'

const debug = _debug('peercalls')

export interface MediaDevice {
  id: string
  name: string
  type: 'audioinput' | 'videoinput'
}

export const enumerateDevices = makeAction(MEDIA_ENUMERATE, async () => {
  const devices = await navigator.mediaDevices.enumerateDevices()

  return devices
  .filter(
    device => device.kind === 'audioinput' || device.kind === 'videoinput')
  .map(device => ({
    id: device.deviceId,
    type: device.kind,
    name: device.label,
  }) as MediaDevice)

})

export type FacingMode = 'user' | 'environment'

export interface DeviceConstraint {
  deviceId: string
}

export interface FacingConstraint {
  facingMode: FacingMode | { exact: FacingMode }
}

export type VideoConstraint = DeviceConstraint | boolean | FacingConstraint
export type AudioConstraint = DeviceConstraint | boolean

export interface GetMediaConstraints {
  video: VideoConstraint
  audio: AudioConstraint
}

declare global {
  interface Navigator {
    webkitGetUserMedia?: typeof navigator.getUserMedia
    mozGetUserMedia?: typeof navigator.getUserMedia
  }
}

async function getUserMedia(
  constraints: MediaStreamConstraints,
): Promise<MediaStream> {
  if (navigator.mediaDevices && navigator.mediaDevices.getUserMedia) {
    return navigator.mediaDevices.getUserMedia(constraints)
  }

  const _getUserMedia: typeof navigator.getUserMedia =
    navigator.getUserMedia ||
    navigator.webkitGetUserMedia ||
    navigator.mozGetUserMedia

  return new Promise<MediaStream>((resolve, reject) => {
    _getUserMedia.call(navigator, constraints, resolve, reject)
  })
}

async function getDisplayMedia(): Promise<MediaStream> {
  const mediaDevices = navigator.mediaDevices as any // eslint-disable-line
  return mediaDevices.getDisplayMedia({video: true, audio: false})
}

export interface MediaVideoConstraintAction {
  type: 'MEDIA_VIDEO_CONSTRAINT_SET'
  payload: VideoConstraint
}

export interface MediaAudioConstraintAction {
  type: 'MEDIA_AUDIO_CONSTRAINT_SET'
  payload: AudioConstraint
}

export function setVideoConstraint(
  payload: VideoConstraint,
): MediaVideoConstraintAction {
  return {
    type: MEDIA_VIDEO_CONSTRAINT_SET,
    payload,
  }
}

export function setAudioConstraint(
  payload: AudioConstraint,
): MediaAudioConstraintAction {
  return {
    type: MEDIA_AUDIO_CONSTRAINT_SET,
    payload,
  }
}

export const play = makeAction('MEDIA_PLAY', async () => {
  const promises = Array
  .from(document.querySelectorAll('video'))
  .filter(video => video.paused)
  .map(video => video.play())
  await Promise.all(promises)
})

export const getMediaStream = makeAction(
  MEDIA_STREAM,
  async (constraints: GetMediaConstraints) => {
    debug('getMediaStream', constraints)
    const payload: AddStreamPayload = {
      stream: await getUserMedia(constraints),
      type: STREAM_TYPE_CAMERA,
      userId: ME,
    }
    return payload
  },
)

export const getDesktopStream = makeAction(
  MEDIA_STREAM,
  async () => {
    debug('getDesktopStream')
    const payload: AddStreamPayload = {
      stream: await getDisplayMedia(),
      type: STREAM_TYPE_DESKTOP,
      userId: ME,
    }
    return payload
  },
)

export type MediaEnumerateAction = AsyncAction<'MEDIA_ENUMERATE', MediaDevice[]>
export type MediaStreamAction = AsyncAction<'MEDIA_STREAM', AddStreamPayload>
export type MediaPlayAction = AsyncAction<'MEDIA_PLAY', void>

export type MediaAction =
  MediaVideoConstraintAction |
  MediaAudioConstraintAction |
  MediaEnumerateAction |
  MediaStreamAction |
  MediaPlayAction
