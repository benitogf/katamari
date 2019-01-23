import React, { Component } from 'react'
import { BrowserRouter as Router, Route } from 'react-router-dom'
import './App.css';

import Boxes from './Boxes'
import Box from './Box'
import Thing from './Thing'

import moment from 'moment'
import Socket from './Socket'
import AppBar from '@material-ui/core/AppBar'
import Toolbar from '@material-ui/core/Toolbar'
import Typography from '@material-ui/core/Typography'
import LinearProgress from '@material-ui/core/LinearProgress'
import SnackbarContent from '@material-ui/core/SnackbarContent'
import Paper from '@material-ui/core/Paper'
import IconButton from '@material-ui/core/IconButton'
import Icon from '@material-ui/core/Icon'
import { Switch, Link } from 'react-router-dom'

const address = 'localhost:8800'

const r404 = () => (<Paper className="paper-container" elevation={1}>
  <SnackbarContent
    style={{
      backgroundColor: '#f1932c',
      maxWidth: 'unset'
    }}
    action={[
      <IconButton
        key="close"
        aria-label="Close"
        color="inherit"
        {...{ to: '/' }}
        component={Link}>
        <Icon>cancel</Icon>
      </IconButton>
    ]}
    message={(
      <Typography component="p" style={{ color: 'white' }}>
        <Icon style={{ verticalAlign: "bottom", color: "#f0cf81" }}>warning</Icon> Could not find this path.
      </Typography>
    )}
  /></Paper>)

class App extends Component {
  constructor(props) {
    super(props)
    const socket = new Socket(
      address + '/time'
    )
    this.state = {
      time: null,
      socket
    };
    socket.onerror = (evt) => {
      // console.info(evt)
      this.setState({
        time: null
      })
    }
    socket.onmessage = (evt) => {
      // console.info(evt)
      this.setState({
        time: socket.samo.parseTime(evt)
      })
    }
  }
  componentWillUnmount() {
    this.state.socket.close()
  }
  render() {
    const { time } = this.state
    return (!time) ? (<LinearProgress />) : (
      <div className="App">
        <AppBar position="sticky" color="default">
          <Toolbar>
            <Typography component="p">
              {moment.unix(time / 1000000000).format('dddd, MMMM Do, Y. LTS')}
            </Typography>
          </Toolbar>
        </AppBar>
        <Router>
          <Switch>
            <Route path="/" exact component={Boxes} />
            <Route
              path="/box/:id"
              render={(props) => <Box {...props} time={this.state.time} />}
            />
            <Route
              path="/thing/:box/:id"
              render={(props) => <Thing {...props} time={this.state.time} />}
            />
            <Route component={r404} />
          </Switch>
        </Router>
      </div>
    )
  }
}

export default App
