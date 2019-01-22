import React, { Component } from 'react'
import Socket from './Socket'
import Typography from '@material-ui/core/Typography'
import LinearProgress from '@material-ui/core/LinearProgress'
import List from '@material-ui/core/List'
import ListItem from '@material-ui/core/ListItem'
import ListItemText from '@material-ui/core/ListItemText'
import Paper from '@material-ui/core/Paper'
import { Link } from 'react-router-dom'

class Box extends Component {
    constructor(props) {
        super(props)
        const socket = new Socket(
            'ws://localhost:8800/mo/boxes/' + props.match.params.id
        )
        this.state = {
            things: null,
            socket
        }

        socket.onopen = (evt) => {
            // console.info(evt)
            socket.put({
                name: "a thing in the box: " + props.match.params.id
            })
        }
        socket.onerror = (evt) => {
            // console.info(evt)
            this.setState({
                things: null
            })
        }
        socket.onmessage = (evt) => {
            // console.info(evt)
            this.setState({
                things: socket.decode(evt)
            })
        }
    }
    componentWillUnmount() {
        this.state.socket.close()
    }
    render() {
        const { things } = this.state
        return (!things) ? (<LinearProgress />) : (
            <Paper className="paper-container" elevation={1}>
                {(() => things.length !== 0 ? (
                    <List component="nav" className="list">
                        {things.map((thing) =>
                            <ListItem
                                {...{ to: '/thing/' + this.props.match.params.id + '/' + thing.index }}
                                component={Link}
                                key={thing.index}
                                button>
                                <ListItemText primary={thing.data.name + ' (' + thing.index + ')'} />
                            </ListItem>
                        )}
                    </List>
                ) : (
                        <Typography className="paper-content" component="h2">
                            There are no things yet.
                        </Typography>
                    ))()}
            </Paper>
        )
    }
}

export default Box