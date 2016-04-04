const React = require('react');
const Transition = require('react-addons-css-transition-group');
const notificationsStore = require('../store/notificationsStore.js');

function notifications(props) {
  let notifs = notificationsStore.getNotifications(props.max || 10);

  let notificationElements = notifs.map(notif => {
    return (
      <div className={notif.type + ' notification'} key={notif._id}>
        {notif.message}
      </div>
    );
  });

  return (
    <div className="notifications">
      <Transition
        transitionEnterTimeout={200}
        transitionLeaveTimeout={100}
        transitionName="fade"
      >
        {notificationElements}
      </Transition>
    </div>
  );

}

module.exports = notifications;
