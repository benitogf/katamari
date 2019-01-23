import React, { Component } from 'react'
import Socket from './Socket'
import LinearProgress from '@material-ui/core/LinearProgress'
import Typography from '@material-ui/core/Typography'
import Paper from '@material-ui/core/Paper'

const address = 'localhost:8800'

class Thing extends Component {
    constructor(props) {
        super(props)
        const socket = new Socket(
            address + '/sa/boxes/' + props.match.params.box + '/' + props.match.params.id
        )
        this.state = {
            thing: null,
            socket
        }

        socket.onerror = (evt) => {
            // console.info(evt)
            this.setState({
                thing: null
            })
        }
        socket.onmessage = (evt) => {
            // console.info(evt)
            this.setState({
                thing: socket.samo.decode(evt)
            })
        }
    }
    componentWillUnmount() {
        this.state.socket.close()
    }
    render() {
        const { thing } = this.state
        return (!thing) ? (<LinearProgress />) : (
            <Paper className="paper-container" elevation={1}>
                <Typography className="paper-content" component="h2">
                    {JSON.stringify(thing)}
                </Typography>
            </Paper>
        )
    }
}

export default Thing