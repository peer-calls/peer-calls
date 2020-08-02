export interface WorkerMessageEvent<T> {
  data: T
}

export interface WebWorker<RecvType, SendType> {
  onmessage: ((e: WorkerMessageEvent<RecvType>) => void) | null
  postMessage: (data: SendType, transfer: Transferable[]) => void
}

