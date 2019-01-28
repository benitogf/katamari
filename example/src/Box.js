import React, { Component } from 'react'
import Samo from 'samo-js-client'
import Typography from '@material-ui/core/Typography'
import LinearProgress from '@material-ui/core/LinearProgress'
import List from '@material-ui/core/List'
import ListItem from '@material-ui/core/ListItem'
import ListItemText from '@material-ui/core/ListItemText'
import Paper from '@material-ui/core/Paper'
import { Link } from 'react-router-dom'

const address = 'localhost:8800'

class Box extends Component {
    constructor(props) {
        super(props)
        const box = new Samo(
            address + '/mo/boxes/' + props.match.params.id
        )
        this.state = {
            things: null,
            socket: box
        }

        box.onopen = (evt) => {
            // console.info(evt)
            box.set({
                name: "a thing in the box: " + props.match.params.id
            })
        }
        box.onerror = (evt) => {
            // console.info(evt)
            this.setState({
                things: null
            })
        }
        box.onmessage = (evt) => {
            // console.info(evt)
            this.setState({
                things: box.decode(evt)
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