import PropTypes from 'prop-types'
import React from 'react'
import * as constants from '../constants.js'

import {
  StyleSheet,
  View,
  Text,
  Image,
  Dimensions
} from 'react-native'

import {
  withScriptjs,
  withGoogleMap,
  GoogleMap,
  Marker
} from 'react-google-maps'

const screen = Dimensions.get('window')

const GoogleMapContainer = withScriptjs(withGoogleMap(props => <GoogleMap {...props} />));

export default class Map extends React.PureComponent {
  static propTypes = {
    positions: PropTypes.object.isRequired
  }
  render () {
    const { positions } = this.props
    const defaultCenter = Object.keys(positions).length
      ? {
        lat: positions[Object.keys(positions)[0]].coords.latitude,
        lng: positions[Object.keys(positions)[0]].coords.longitude
      }
      : null

    return (
      <div className="drawer-content">
        <View style={styles.container}>
          <View style={styles.map}>

            {defaultCenter ? (
              <GoogleMapContainer
                googleMapURL={constants.GOOGLE_MAPS_API_URL}
                loadingElement={<div style={{ height: '100%' }} />}
                containerElement={<div
                  style={{ height: `${screen.height - 52}px` }}
                  // 52px is the height of the `.drawer-header`
                />}
                mapElement={<div style={{ height: '100%' }} />}
                defaultCenter={defaultCenter}
                defaultZoom={8}
              >
                {this.createMarkers()}
              </GoogleMapContainer>
            ) : (
              <div className="drawer-empty">
                <span className="drawer-empty-icon icon icon-room" />
                <div className="drawer-empty-message">No Positions</div>
              </div>
            )}

          </View>
          <View pointerEvents="none" style={styles.members}>
            {this.createMembers()}
          </View>
        </View>
      </div>
    )
  }

  createMarkers() {
    const { positions } = this.props
    const members = Object.keys(positions)
    return Object.values(positions).map((position, index) => {
      const id = members[index]
      const { coords } = position
      return (
        <Marker
          key={id}
          title={id}
          position={{ lat: coords.latitude, lng: coords.longitude }}
          icon={{ url: `http://maps.google.com/mapfiles/kml/pal3/icon${index}.png` }}
        />
      )
    })
  }

  createMembers() {
    const { positions } = this.props
    const members = Object.keys(positions)
    return members.map((id, index) => {
      return (
        <View key={id} style={styles.member}>
          <Image
            source={`http://maps.google.com/mapfiles/kml/pal3/icon${index}.png`}
            style={styles.avatar}
          />
          <Text style={styles.memberName}>{id}</Text>
        </View>
      )
    })
  }
}

const colors = [
  '#e6194b', '#3cb44b', '#ffe119', '#0082c8',
  '#f58231', '#911eb4', '#46f0f0', '#f032e6',
  '#d2f53c', '#fabebe', '#008080', '#e6beff',
  '#aa6e28', '#fffac8', '#800000', '#aaffc3',
  '#808000', '#ffd8b1', '#000080'
]

const getColor = function(index = 0) {
  const color = colors[index]
  index = ++index % colors.length
  return color
}

const styles = StyleSheet.create({
  container: {
    ...StyleSheet.absoluteFillObject,
    justifyContent: 'flex-end',
    alignItems: 'center',
  },
  map: {
    ...StyleSheet.absoluteFillObject,
  },
  bubble: {
    flex: 1,
    backgroundColor: 'rgba(255,255,255,0.7)',
    paddingHorizontal: 18,
    paddingVertical: 12,
    borderRadius: 20,
    marginRight: 20,
  },
  latlng: {
    width: 200,
    alignItems: 'stretch',
  },
  button: {
    width: 80,
    paddingHorizontal: 12,
    alignItems: 'center',
    marginHorizontal: 10,
  },
  buttonContainer: {
    flexDirection: 'row',
    marginVertical: 20,
    backgroundColor: 'transparent',
  },
  members: {
    flexDirection: 'column',
    justifyContent: 'flex-start',
    alignItems: 'flex-start',
    width: '100%',
    paddingHorizontal: 10,
  },
  member: {
    flexDirection: 'row',
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: 'rgba(255,255,255,1)',
    borderRadius: 20,
    height: 30,
    marginBottom: 10,
  },
  memberName: {
    marginHorizontal: 10,
  },
  avatar: {
    height: 30,
    width: 30,
    borderRadius: 15,
  }
})
