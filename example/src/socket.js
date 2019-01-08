// https://github.com/daviddoran/typescript-reconnecting-websocket/blob/master/reconnecting-websocket.ts
import { Base64 } from 'js-base64'

class ReconnectingWebSocket {
    /**
     * Setting this to true is the equivalent of setting all instances of ReconnectingWebSocket.debug to true.
     */
    debugAll = false;
    // These can be altered by calling code
    debug = false;

    // Time to wait before attempting reconnect (after close)
    reconnectInterval = 1000;
    // Time to wait for WebSocket to open (before aborting and retrying)
    timeoutInterval = 2000;

    // Should only be used to read WebSocket readyState
    readyState;

    // Whether WebSocket was forced to close by this client
    forcedClose = false;
    // Whether WebSocket opening timed out
    timedOut = false;

    // List of WebSocket sub-protocols
    protocols = [];

    // The underlying WebSocket
    ws;
    url;

    // Set up the default 'noop' event handlers
    onopen = (ev) => {};
    onclose = (ev) => {};
    onconnecting = (ev) => {};
    onmessage = (ev) => {};
    onerror = (ev) => {};

    constructor(url, protocols = []) {
        this.url = url;
        this.protocols = protocols;
        this.readyState = WebSocket.CONNECTING;
        this.connect(false);
    }

    connect(reconnectAttempt) {
        this.ws = new WebSocket(this.url, this.protocols);

        this.onconnecting();
        this.log('ReconnectingWebSocket', 'attempt-connect', this.url);

        var localWs = this.ws;
        var timeout = setTimeout(
          () => {
            this.log('ReconnectingWebSocket', 'connection-timeout', this.url);
            this.timedOut = true;
            localWs.close();
            this.timedOut = false;
          },
          this.timeoutInterval);

        this.ws.onopen = (event) => {
            clearTimeout(timeout);
            this.log('ReconnectingWebSocket', 'onopen', this.url);
            this.readyState = WebSocket.OPEN;
            reconnectAttempt = false;
            this.onopen(event);
        };

        this.ws.onclose = (event) => {
            clearTimeout(timeout);
            // this.ws = null;
            if (this.forcedClose) {
                this.readyState = WebSocket.CLOSED;
                this.onclose(event);
            } else {
                this.readyState = WebSocket.CONNECTING;
                this.onconnecting();
                if (!reconnectAttempt && !this.timedOut) {
                    this.log('ReconnectingWebSocket', 'onclose', this.url);
                    this.onclose(event);
                }
                setTimeout(
                  () => {
                    this.connect(true);
                  },
                  this.reconnectInterval);
            }
        };
        this.ws.onmessage = (event) => {
            this.log('ReconnectingWebSocket', 'onmessage', this.url, event.data);
            this.onmessage(event);
        };
        this.ws.onerror = (event) => {
            this.log('ReconnectingWebSocket', 'onerror', this.url, event);
            this.onerror(event);
        };
    }

    send(data) {
        if (this.ws) {
            this.log('ReconnectingWebSocket', 'send', this.url, data);
            return this.ws.send(data);
        } else {
            throw new Error('INVALID_STATE_ERR : Pausing to reconnect websocket');
        }
    }

    del(index) {
        if (this.ws) {
            this.log('ReconnectingWebSocket', 'del', this.url, index);
            return this.ws.send(JSON.stringify({
                op: "DEL",
                index
               }));
        } else {
            throw new Error('INVALID_STATE_ERR : Pausing to reconnect websocket');
        }
    }

    put(data, index) {
        if (this.ws) {
            this.log('ReconnectingWebSocket', 'put', this.url, data);
            return this.ws.send(this.encode(data, index));
        } else {
            throw new Error('INVALID_STATE_ERR : Pausing to reconnect websocket');
        }
    }

    /**
     * Returns boolean, whether websocket was FORCEFULLY closed.
     */
    close(force, url) {
        if (url !== undefined && force) {
            this.url = url;
        }
        if (this.ws) {
            this.forcedClose = !force;
            this.ws.close();
            return true;
        }
        return false;
    }

    /**
     * Additional public API method to refresh the connection if still open (close, re-open).
     * For example, if the app suspects bad data / missed heart beats, it can try to refresh.
     *
     * Returns boolean, whether websocket was closed.
     */
    refresh() {
        if (this.ws) {
            this.ws.close();
            return true;
        }
        return false;
    }

    decode(evt) {
        const msg = Base64.decode(JSON.parse(evt.data).data)
        const data =  msg !== "" ? JSON.parse(msg) : {created:0, updated:0, index: "", data: "e30="}
        const mode = evt.currentTarget.url.replace("ws://localhost:8800/", "").split("/")[0]
        return (mode === "sa") ? Object.assign(data, {data: JSON.parse(Base64.decode(data["data"]))}) :
            data.map((obj) => {
            obj['data'] = JSON.parse(Base64.decode(obj["data"]))
            return obj
            })
    }
      
    encode = (data, index) => JSON.stringify({
        data: Base64.encode(JSON.stringify(data)),
        index
    })

    parseTime = (evt) => new Date(parseInt(JSON.parse(evt.data).data)/1000000)

    log(...args) {
        if (this.debug || ReconnectingWebSocket.debugAll) {
            console.debug.apply(console, args);
        }
    }
}

export default ReconnectingWebSocket;