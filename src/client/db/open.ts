import { promisify } from './promisify'
import { Upgrade, upgrade } from './upgrade'

export async function open(
  name: string,
  version: number,
  doUpgrade: Upgrade = upgrade,
) {
  const request = window.indexedDB.open(name, version)
  request.onupgradeneeded = doUpgrade
  return promisify(request)
}
