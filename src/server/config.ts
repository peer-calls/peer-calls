import cfg, { IConfig } from 'config'

export type ICEServer = {
  url: string
  urls: string[] | string
  auth: 'secret'
  username: string
  secret: string
} | {
  url: string
  urls: string[] | string
  auth: undefined
  username: string
  credential: string
}

export interface Config {
  baseUrl: string
  iceServers: ICEServer[]
  ssl: {
    cert: string
    key: string
  }
}

export const config = cfg as IConfig & Config
