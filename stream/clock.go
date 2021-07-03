package stream

import (
	"time"

	"github.com/gorilla/websocket"
)

// BroadcastClock sends time to all the subscribers
func (sm *Stream) BroadcastClock(data string) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	connections := sm.pools[0].connections

	for _, client := range connections {
		sm.WriteClock(client, data)
	}
}

// WriteClock sends time to a subscriber
func (sm *Stream) WriteClock(client *Conn, data string) {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	client.conn.SetWriteDeadline(time.Now().Add(timeout))
	err := client.conn.WriteMessage(websocket.BinaryMessage, []byte(data))
	if err != nil {
		client.conn.Close()
		sm.Console.Log("writeTimeStreamErr: ", err)
	}
}
