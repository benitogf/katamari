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

class Boxes extends Component {
    constructor(props) {
        super(props)
        const boxes = new Samo(
            address + '/mo/boxes'
        )
        this.state = {
            boxes: null,
            socket: boxes
        }

        boxes.onopen = (evt) => {
            // console.info(evt)
            boxes.set({
                name: "a box"
            })
        }
        boxes.onerror = (evt) => {
            // console.info(evt)
            this.setState({
                boxes: null
            })
        }
        boxes.onmessage = (evt) => {
            // console.info(evt)
            this.setState({
                boxes: boxes.decode(evt)
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