'use strict'
import _debug from 'debug'
import { ServerSocket, TypedIO } from '../shared'
import { Store } from './store'

const debug = _debug('peercalls:socket')

export interface Stores {
  userIdBySocketId: Store
  socketIdByUserId: Store
}

export default function handleSocket(
  socket: ServerSocket,
  io: TypedIO,
  stores: Stores,
) {
  socket.once('disconnect', async () => {
    const userId = await stores.userIdBySocketId.get(socket.id)
    if (userId) {
      await Promise.all([
        stores.userIdBySocketId.remove(socket.id),
        stores.socketIdByUserId.remove(userId),
      ])
    }
  })

  socket.on('signal', async payload => {
    // debug('signal: %s, payload: %o', socket.userId, payload)
    const socketId = await stores.socketIdByUserId.get(payload.userId)
    const userId = await stores.userIdBySocketId.get(socket.id)
    if (socketId) {
      io.to(socketId).emit('signal', {
        userId,
        signal: payload.signal,
      })
    }
  })

  socket.on('ready', async payload => {
    const { userId, room } = payload
    debug('ready: %s, room: %s', userId, room)
    // no need to leave rooms because there will be only one room for the
    // duration of the socket connection
    await Promise.all([
      stores.socketIdByUserId.set(userId, socket.id),
      stores.userIdBySocketId.set(socket.id, userId),
    ])
    socket.join(room)

    const users = await getUsers(room)

    debug('ready: %s, room: %s, users: %o', userId, room, users)

    io.to(room).emit('users', {
      initiator: userId,
      users,
    })
  })

  async function getUsers (room: string) {
    const socketIds = await getClientsInRoom(room)
    const userIds = await stores.userIdBySocketId.getMany(socketIds)
    return socketIds.map((socketId, i) => ({
      socketId,
      userId: userIds[i],
    }))
  }

  async function getClientsInRoom(room: string): Promise<string[]> {
    return new Promise((resolve, reject) => {
      io.in(room).clients((err: Error, clients: string[]) => {
        if (err) {
          reject(err)
        } else {
          resolve(clients)
        }
      })
    })
  }
}

