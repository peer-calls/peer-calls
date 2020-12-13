export interface QueryString {
  [key: string]: string
}

export function parseQueryString(qs: string): QueryString {
  return qs.substring(1).split('&').reduce((obj, kv) => {
    const [k, v] = kv.split('=', 2)
    obj[decodeURIComponent(k)] = decodeURIComponent(v)
    return obj
  }, {} as QueryString)
}
