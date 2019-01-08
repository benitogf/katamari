import React, { Component } from 'react';
import './App.css';
import Socket from './socket'
import List from '@material-ui/core/List';
import ListItem from '@material-ui/core/ListItem';
import ListItemText from '@material-ui/core/ListItemText';
import Paper from '@material-ui/core/Paper';
import Typography from '@material-ui/core/Typography';
import Grid from '@material-ui/core/Grid';

const testMode = "mo" // TODO: interactive mode toggle

class App extends Component {
  constructor(props) {
    super(props);
    const socket = new Socket(
      'ws://localhost:8800/' + testMode + '/test'
    )
    const time = new Socket(
      'ws://localhost:8800/time'
    )
    this.state = {
      list: null,
      time: null,
      socket
    };

    time.onopen = (evt) => {
      console.info(evt)
    }
    time.onclose = (evt) => {
      console.info(evt)
    }
    time.onerror = (evt) => {
      console.info(evt)
    }
    time.onmessage = (evt) => {
      // console.info(evt)
      this.setState({
        time: socket.parseTime(evt).toString()
      })
    }
    
    socket.onopen = (evt) => {
      console.info(evt)
      socket.put({
        test: 1234
      })
    }
    
    socket.onclose = (evt) => {
      console.info(evt)
    }
    
    socket.onerror = (evt) => {
      console.info(evt)
    }
    
    socket.onmessage = (evt) => {
      console.info(evt)
      this.setState({
        data: socket.decode(evt),
      })
    }
  }

  render() {
    if (!this.state.data || !this.state.time) {
      return (
        <div className="App">
          <header className="App-header">
            loading...
          </header>
        </div>
      )
    } else {
      return (
        <div className="App">
          <header className="App-header">
          <Paper style={{
             padding: "20px",
            }} elevation={1}>
            <Grid 
              container
              direction="row"
              justify="flex-end"
              alignItems="flex-end"
              spacing={0}>
              <Grid item>
                <Typography component="p">
                  {this.state.time}
                </Typography>
              </Grid>
            </Grid>
          </Paper>
          {(() => (Array.isArray(this.state.data)) ?
              (
                <List component="nav">
                {this.state.data.map((event) =>
                  <ListItem key={event.index} button onClick={()=>
                    this.state.socket.del(event.index)
                  }>
                    <ListItemText primary={event.index} />
                  </ListItem>
                )}          
              </List>
              ) : (
                <Typography component="p">
                  {JSON.stringify(this.state.data)}
                </Typography>                
              ))()}
          </header>
        </div>
      );
    }
  }
}

export default App;
