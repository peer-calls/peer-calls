import { config } from '../config'
import * as turn from '../turn'
import { Router } from 'express'
import { v4 } from 'uuid'

const router = Router()

const BASE_URL: string = config.baseUrl
const cfgIceServers = config.iceServers

router.post('/', (req, res) => {
  const callId = req.body.call ? encodeURIComponent(req.body.call) : v4()
  res.redirect(`${BASE_URL}/call/${callId}`)
})

router.get('/:callId', (req, res) => {
  const iceServers = turn.processServers(cfgIceServers)
  res.render('call', {
    callId: encodeURIComponent(req.params.callId),
    userId: v4(),
    nickname: req.headers['x-forwarded-user'] || '',
    iceServers,
  })
})

export default router
