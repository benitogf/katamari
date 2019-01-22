import React, { Component } from 'react'
import Socket from './Socket'
import Typography from '@material-ui/core/Typography'
import LinearProgress from '@material-ui/core/LinearProgress'
import List from '@material-ui/core/List'
import ListItem from '@material-ui/core/ListItem'
import ListItemText from '@material-ui/core/ListItemText'
import Paper from '@material-ui/core/Paper'
import { Link } from 'react-router-dom'

class Boxes extends Component {
    constructor(props) {
        super(props)
        const socket = new Socket(
            'ws://localhost:8800/mo/boxes'
        )
        this.state = {
            boxes: null,
            socket
        }

        socket.onopen = (evt) => {
            // console.info(evt)
            socket.put({
                name: "a box"
            })
        }
        socket.onerror = (evt) => {
            // console.info(evt)
            this.setState({
                boxes: null
            })
        }
        socket.onmessage = (evt) => {
            // console.info(evt)
            this.setState({
                boxes: socket.decode(evt)
            })
        }
    }
    componentWillUnmount() {
        this.state.socket.close()
    }
    render() {
        const { boxes } = this.state
        return (!boxes) ? (<LinearProgress />) : (
            <Paper className="paper-container" elevation={1}>
                {(() => boxes.length !== 0 ? (
                    <List component="nav" className="list">
                        {boxes.map((box) =>
                            <ListItem
                                {...{ to: '/box/' + box.index }}
                                component={Link}
                                key={box.index}
                                button>
                                <ListItemText primary={box.data.name + ' (' + box.index + ')'} />
                            </ListItem>
                        )}
                    </List>
                ) : (
                        <Typography className="paper-content" component="h2">
                            There are no boxes yet.
                        </Typography>
                    ))()}
            </Paper>
        )
    }
}

export default Boxes