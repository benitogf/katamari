package stream

import "github.com/gorilla/websocket"

// BroadcastTime sends time to all the subscribers
func (sm *Pools) BroadcastTime(data string) {
	sm.mutex.RLock()
	connections := sm.Pools[0].connections
	sm.mutex.RUnlock()

	for _, client := range connections {
		go sm.WriteTime(client, data)
	}
}

// WriteTime sends time to a subscriber
func (sm *Pools) WriteTime(client *Conn, data string) {
	client.mutex.Lock()
	err := client.conn.WriteMessage(websocket.BinaryMessage, []byte(data))
	client.mutex.Unlock()
	if err != nil {
		sm.Console.Log("writeStreamErr: ", err)
	}
}
