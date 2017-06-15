import * as NotifyActions from '../actions/NotifyActions.js'
import * as CallActions from '../actions/CallActions.js'
import App from '../components/App.js'
import React from 'react'
import { bindActionCreators } from 'redux'
import { connect } from 'react-redux'
import peers from '../peer/peers.js'

function mapStateToProps(state) {
  return {
    streams: state.streams.all,
    alerts: state.alerts,
    notifications: state.notifications,
    active: state.streams.active
  }
}

function mapDispatchToProps(dispatch) {
  return {
    activate: bindActionCreators(CallActions.activateStream, dispatch),
    dismiss: bindActionCreators(NotifyActions.dismiss, dispatch),
    init: bindActionCreators(CallActions.init, dispatch),
    notify: bindActionCreators(NotifyActions.info, dispatch)
  }
}

export default connect(mapStateToProps, mapDispatchToProps)(App)
