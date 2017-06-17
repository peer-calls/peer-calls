import * as CallActions from '../actions/CallActions.js'
import * as NotifyActions from '../actions/NotifyActions.js'
import * as StreamActions from '../actions/StreamActions.js'
import App from '../components/App.js'
import { bindActionCreators } from 'redux'
import { connect } from 'react-redux'

function mapStateToProps (state) {
  return {
    streams: state.streams.all,
    alerts: state.alerts,
    notifications: state.notifications,
    active: state.streams.active
  }
}

function mapDispatchToProps (dispatch) {
  return {
    activate: bindActionCreators(StreamActions.activateStream, dispatch),
    dismissAlert: bindActionCreators(NotifyActions.dismissAlert, dispatch),
    init: bindActionCreators(CallActions.init, dispatch),
    notify: bindActionCreators(NotifyActions.info, dispatch)
  }
}

export default connect(mapStateToProps, mapDispatchToProps)(App)
