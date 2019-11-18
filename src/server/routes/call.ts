import { config } from '../config'
import * as turn from '../turn'
import { Router } from 'express'
import { v4 } from 'uuid'

const router = Router()

const BASE_URL: string = config.baseUrl
const cfgIceServers = config.iceServers

router.get('/', (req, res) => {
  res.redirect(`${BASE_URL}/call/${v4()}`)
})

router.get('/:callId', (req, res) => {
  const iceServers = turn.processServers(cfgIceServers)
  res.render('call', {
    callId: encodeURIComponent(req.params.callId),
    iceServers,
  })
})

export default router
