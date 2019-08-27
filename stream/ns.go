package stream

import (
	"strconv"
	"sync"

	"github.com/benitogf/nsocket"
)

// Nconn extends the named socket connection with a mutex
type Nconn struct {
	mutex sync.Mutex
	conn  *nsocket.Client
}

// CloseNs connection
func (sm *Pools) CloseNs(client *Nconn) {
	// auxiliar clients array
	na := []*Nconn{}

	// loop to remove this client
	sm.mutex.Lock()
	poolIndex := sm.findPool(client.conn.Path, client.conn.Path)
	for _, v := range sm.Pools[poolIndex].nconnections {
		if v != client {
			na = append(na, v)
		}
	}

	// replace clients array with the auxiliar
	sm.Pools[poolIndex].nconnections = na
	sm.mutex.Unlock()
	go sm.OnUnsubscribe(client.conn.Path)
	client.conn.Close()
}

// OpenNs connection for a key
func (sm *Pools) OpenNs(nsClient *nsocket.Client) (*Nconn, int) {
	client := &Nconn{
		conn:  nsClient,
		mutex: sync.Mutex{},
	}

	sm.mutex.Lock()
	poolIndex := sm.findPool(client.conn.Path, client.conn.Path)
	if poolIndex == -1 {
		// create a pool
		sm.Pools = append(
			sm.Pools,
			&Pool{
				Key:          client.conn.Path,
				Filter:       client.conn.Path,
				connections:  []*Conn{},
				nconnections: []*Nconn{client}})
		poolIndex = len(sm.Pools) - 1
	} else {
		// use existing pool
		sm.Pools[poolIndex].nconnections = append(
			sm.Pools[poolIndex].nconnections,
			client)
	}
	sm.Console.Log("nconnections["+client.conn.Path+"]: ", len(sm.Pools[poolIndex].nconnections))
	sm.mutex.Unlock()

	return client, poolIndex
}

// WriteNs will write data to a ns connection
func (sm *Pools) WriteNs(client *Nconn, data string, snapshot bool) {
	client.mutex.Lock()
	err := client.conn.Write("{" +
		"\"snapshot\": " + strconv.FormatBool(snapshot) + "," +
		"\"data\": \"" + data + "\"" +
		"}")
	client.mutex.Unlock()
	if err != nil {
		sm.Console.Log("writeStreamErr: ", err)
	}
}

// ReadNs will keep alive the ns connection
func (sm *Pools) ReadNs(client *Nconn) {
	for {
		_, err := client.conn.Read()
		if err != nil {
			sm.Console.Err("readNsError["+client.conn.Path+"]", err)
			break
		}
	}
	sm.CloseNs(client)
}
