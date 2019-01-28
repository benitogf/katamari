import React, { Component } from 'react'
import LinearProgress from '@material-ui/core/LinearProgress'
import Typography from '@material-ui/core/Typography'
import Paper from '@material-ui/core/Paper'
import Samo from 'samo-js-client'

const address = 'localhost:8800'

class Thing extends Component {
    constructor(props) {
        super(props)
        const thing = new Samo(
            address + '/sa/boxes/' + props.match.params.box + '/' + props.match.params.id
        )
        this.state = {
            thing: null,
            socket: thing
        }

        thing.onerror = (evt) => {
            // console.info(evt)
            this.setState({
                thing: null
            })
        }
        thing.onmessage = (evt) => {
            // console.info(evt)
            this.setState({
                thing: thing.decode(evt)
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