package stream

import "github.com/gorilla/websocket"

// BroadcastTime sends time to all the subscribers
func (sm *Pools) BroadcastTime(data string) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	connections := sm.Pools[0].connections

	for _, client := range connections {
		sm.WriteTime(client, data)
	}
}

// WriteTime sends time to a subscriber
func (sm *Pools) WriteTime(client *Conn, data string) {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	err := client.conn.WriteMessage(websocket.BinaryMessage, []byte(data))
	if err != nil {
		sm.Console.Log("writeTimeStreamErr: ", err)
	}
}
