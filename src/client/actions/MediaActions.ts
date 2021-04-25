import _debug from 'debug'
import { AsyncAction, makeAction } from '../async'
import { audioProcessor } from '../audio'
import { DEVICE_DEFAULT_ID, DEVICE_DISABLED_ID, MEDIA_DEVICE_ID, MEDIA_DEVICE_TOGGLE, MEDIA_ENUMERATE, MEDIA_SIZE_CONSTRAINT, MEDIA_STREAM, MEDIA_TRACK, MEDIA_TRACK_ENABLE } from '../constants'
import { MediaStream } from '../window'
import { AddLocalStreamPayload, StreamType, StreamTypeCamera, StreamTypeDesktop } from './StreamActions'

const debug = _debug('peercalls')

export type MediaKind = 'audio' | 'video'

export const MediaKindVideo: MediaKind = 'video'
export const MediaKindAudio: MediaKind = 'audio'

export interface MediaDevice {
  id: string
  name: string
  type: 'audioinput' | 'videoinput'
}

const getUserMediaFail = (
  constraints: MediaStreamConstraints,
  resolve: () => void,
  reject: (err: Error) => void,
) => {
  reject(new Error(
    'No API to retrieve media stream. This can happen if you ' +
    'are using an old browser, or the application is not using HTTPS'))
}

export const enumerateDevices = makeAction(MEDIA_ENUMERATE, async () => {
  let stream: MediaStream
  try {
    stream = await getUserMedia({ audio: true, video: true })
  } catch (err) {
    stream = new MediaStream()
  }

  let devices: MediaDeviceInfo[]
  try {
    devices = await navigator.mediaDevices.enumerateDevices()
  } finally {
    stream.getTracks().forEach(track => track.stop())
  }

  const mappedDevices = devices
  .map(device => ({
    id: device.deviceId,
    type: device.kind,
    name: device.label,
  }) as MediaDevice)

  return mappedDevices
})

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
    navigator.mozGetUserMedia ||
    getUserMediaFail

  return new Promise<MediaStream>((resolve, reject) => {
    _getUserMedia.call(navigator, constraints, resolve, reject)
  })
}

export interface DisplayMediaConstraints {
  audio: boolean
  video: boolean
}

const defaultDisplayMediaConstraints: DisplayMediaConstraints = {
  audio: true,
  video: false,
}

async function getDisplayMedia(
  constraints: DisplayMediaConstraints,
): Promise<MediaStream> {
  const mediaDevices = navigator.mediaDevices as any // eslint-disable-line
  return mediaDevices.getDisplayMedia(constraints)
}

export interface SizeConstraint {
  width: number
  height: number
}

export interface MediaSizeConstraintAction {
  type: 'MEDIA_SIZE_CONSTRAINT'
  payload: SizeConstraint | null
}

export interface DeviceId {
  kind: MediaKind
  deviceId: string
}

export interface MediaDeviceIdAction {
  type: 'MEDIA_DEVICE_ID'
  payload: DeviceId
}

export interface MediaDeviceToggle {
  kind: MediaKind
  enabled: boolean
}

export interface MediaDeviceToggleAction {
  type: 'MEDIA_DEVICE_TOGGLE'
  payload: MediaDeviceToggle
}

export function toggleDevice(
  payload: MediaDeviceToggle,
): MediaDeviceToggleAction {
  return {
    type: MEDIA_DEVICE_TOGGLE,
    payload,
  }
}

export function setSizeConstraint(
  payload: SizeConstraint | null,
): MediaSizeConstraintAction {
  return {
    type: MEDIA_SIZE_CONSTRAINT,
    payload,
  }
}

export function setDeviceId(
  payload: DeviceId,
): MediaDeviceIdAction {
  return {
    type: MEDIA_DEVICE_ID,
    payload,
  }
}

export function setDeviceIdOrDisable(
  payload: DeviceId,
): MediaDeviceIdAction | MediaDeviceToggleAction {
  if (payload.deviceId === DEVICE_DISABLED_ID) {
    return toggleDevice({
      kind: payload.kind,
      enabled: false,
    })
  }

  return setDeviceId(payload)
}

export const play = makeAction('MEDIA_PLAY', async () => {
  const promises = Array
  .from(document.querySelectorAll('video'))
  .filter(video => video.paused)
  .map(video => video.play())
  await Promise.all(promises)
})

export type GetMediaTrackParams = {
  kind: MediaKind
  constraint: MediaTrackConstraints | false
}

export interface MediaTrackPayload {
  kind: MediaKind
  track: MediaStreamTrack | undefined
  type: StreamType
}

export const getMediaTrack = makeAction(
  MEDIA_TRACK,
  async (params: GetMediaTrackParams) => {
    const payload: MediaTrackPayload = {
      kind: params.kind,
      track: undefined,
      type: StreamTypeCamera,
    }
    if (!params.constraint) {
      return payload
    }
    if (params.kind === 'audio') {
      const mediaStream = await getUserMedia({
        audio: params.constraint,
        video: false,
      })
      payload.track = mediaStream.getAudioTracks()[0]
    } else {
      const mediaStream = await getUserMedia({
        audio: false,
        video: params.constraint,
      })
      payload.track = mediaStream.getVideoTracks()[0]
    }
    return payload
  },
)

export interface MediaTrackEnablePayload {
  kind: MediaKind
  type: StreamType
}

export interface MediaTrackEnableAction {
  type: 'MEDIA_TRACK_ENABLE'
  payload: MediaTrackEnablePayload
}

// Enables (unmutes) the current desktop A/V track
export function enableMediaTrack(kind: MediaKind): MediaTrackEnableAction {
  return {
    payload: {
      kind,
      type: StreamTypeCamera,
    },
    type: MEDIA_TRACK_ENABLE,
  }
}

export const getMediaStream = makeAction(
  MEDIA_STREAM,
  async (constraints: MediaStreamConstraints) => {
    // Need to init audioProcessor on user action (e.g. click).
    await audioProcessor.init()

    if (!constraints.audio && !constraints.video) {
      const payload: AddLocalStreamPayload = {
        stream: new MediaStream(),
        type: StreamTypeCamera,
      }
      return payload
    }
    debug('getMediaStream', constraints)
    const payload: AddLocalStreamPayload = {
      stream: await getUserMedia(constraints),
      type: StreamTypeCamera,
    }
    return payload
  },
)

export const getDesktopStream = makeAction(
  MEDIA_STREAM,
  async (
    constraints: DisplayMediaConstraints = defaultDisplayMediaConstraints,
  ) => {
    debug('getDesktopStream')
    const payload: AddLocalStreamPayload = {
      stream: await getDisplayMedia(constraints),
      type: StreamTypeDesktop,
    }
    return payload
  },
)

// getDeviceId is a helper for figuring out the correct device ID.
export function getDeviceId(
  enabled: boolean,
  constraint: MediaTrackConstraints,
): string {
  if (!enabled) {
    return DEVICE_DISABLED_ID
  }

  if (typeof constraint.deviceId !== 'string') {
    return DEVICE_DEFAULT_ID
  }

  return constraint.deviceId
}

export function getTracksByKind(stream: MediaStream, kind: MediaKind) {
  return kind === 'video' ? stream.getVideoTracks() : stream.getAudioTracks()
}

export type MediaEnumerateAction = AsyncAction<'MEDIA_ENUMERATE', MediaDevice[]>
export type MediaStreamAction =
  AsyncAction<'MEDIA_STREAM', AddLocalStreamPayload>
export type MediaPlayAction = AsyncAction<'MEDIA_PLAY', void>
export type MediaTrackAction = AsyncAction<'MEDIA_TRACK', MediaTrackPayload>

export type MediaAction =
  MediaDeviceToggleAction |
  MediaDeviceIdAction |
  MediaSizeConstraintAction |
  MediaEnumerateAction |
  MediaStreamAction |
  MediaTrackAction |
  MediaTrackEnableAction |
  MediaPlayAction
