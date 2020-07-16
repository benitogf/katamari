package pivot_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/benitogf/jsonpatch"
	"github.com/benitogf/katamari"
	"github.com/benitogf/katamari/auth"
	"github.com/benitogf/katamari/messages"
	"github.com/benitogf/katamari/objects"
	"github.com/benitogf/katamari/pivot"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/pkg/expect"
	"github.com/stretchr/testify/require"
)

type Thing struct {
	IP string `json:"ip"`
	On bool   `json:"on"`
}

type Settings struct {
	DayEpoch int `json:"startOfDay"`
}

func Register(t *testing.T, server *katamari.Server) string {
	var c auth.Credentials
	// register
	payload := []byte(`{
        "name": "root",
        "account":"root",
				"password": "000",
				"email": "root@root.cc",
				"phone": "555"
    }`)
	req, err := http.NewRequest("POST", "/register", bytes.NewBuffer(payload))
	require.NoError(t, err)
	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response := w.Result()
	require.Equal(t, http.StatusOK, response.StatusCode)
	dec := json.NewDecoder(response.Body)
	err = dec.Decode(&c)
	require.NoError(t, err)
	require.NotEmpty(t, c.Token)
	return c.Token
}

func CreateThing(t *testing.T, server *katamari.Server, token string, ip string) string {
	data := []byte(`{"on": false, "ip":"` + ip + `"}`)
	encodeddata := base64.StdEncoding.EncodeToString(data)
	payload := []byte(`{"data" : "` + encodeddata + `"}`)
	req, err := http.NewRequest("POST", "/things/*", bytes.NewBuffer(payload))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response := w.Result()
	require.Equal(t, http.StatusOK, response.StatusCode)
	// Read return value of Creation
	body, err := ioutil.ReadAll(response.Body)
	require.NoError(t, err)
	var thingObj objects.Object
	err = json.Unmarshal([]byte(body), &thingObj)
	require.NoError(t, err)
	return thingObj.Index
}

func UpdateSettings(t *testing.T, server *katamari.Server, token string, epoch int) {
	data := []byte(`{"startOfDay": ` + strconv.Itoa(epoch) + `}`)
	encodeddata := base64.StdEncoding.EncodeToString(data)
	payload := []byte(`{"data" : "` + encodeddata + `"}`)
	req, err := http.NewRequest("POST", "/settings", bytes.NewBuffer(payload))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response := w.Result()
	require.Equal(t, http.StatusOK, response.StatusCode)
}

func ModifyThing(t *testing.T, server *katamari.Server, token string, current []byte, thingID string, on bool) {
	var thing Thing
	err := json.Unmarshal(current, &thing)
	require.NoError(t, err)
	thing.On = on
	newaux, err := json.Marshal(thing)
	require.NoError(t, err)
	encodeddata := base64.StdEncoding.EncodeToString(newaux)
	payload := []byte(`{"data" : "` + encodeddata + `"}`)

	// posting the data
	req, err := http.NewRequest("POST", "/things/"+thingID, bytes.NewBuffer(payload))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response := w.Result()
	require.Equal(t, http.StatusOK, response.StatusCode)
}

// ReadThing read the thing data and return []byte object
func ReadThing(t *testing.T, server *katamari.Server, token string, thingID string) []byte {
	var aux []byte
	// Get the thing data
	req, err := http.NewRequest("GET", "/things/"+thingID, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)
	response := w.Result()
	require.Equal(t, http.StatusOK, response.StatusCode)

	// Decode data to json object
	body, err := ioutil.ReadAll(response.Body)
	require.NoError(t, err)
	var object objects.Object

	err = json.Unmarshal([]byte(body), &object)
	aux, err = base64.StdEncoding.DecodeString(object.Data)
	require.NoError(t, err)
	return aux
}

func decodeThingsData(message []byte, cache string) ([]Thing, string) {
	var things []Thing
	wsEvent, err := messages.DecodeTest(message)
	expect.Nil(err)
	if wsEvent.Snapshot {
		cache = wsEvent.Data
	} else {
		patch, err := jsonpatch.DecodePatch([]byte(wsEvent.Data))
		expect.Nil(err)
		modifiedBytes, err := patch.Apply([]byte(cache))
		expect.Nil(err)
		cache = string(modifiedBytes)
	}

	var objects []objects.Object
	err = json.Unmarshal([]byte(cache), &objects)
	expect.Nil(err)
	for i := range objects {
		aux, err := base64.StdEncoding.DecodeString(objects[i].Data)
		expect.Nil(err)
		var thing Thing
		err = json.Unmarshal(aux, &thing)
		expect.Nil(err)
		things = append(things, thing)
	}

	return things, cache
}

func decodeSettingsData(message []byte, cache string) (Settings, string) {
	var settings Settings
	wsEvent, err := messages.DecodeTest(message)
	expect.Nil(err)
	if wsEvent.Snapshot {
		cache = wsEvent.Data
	} else {
		patch, err := jsonpatch.DecodePatch([]byte(wsEvent.Data))
		expect.Nil(err)
		modifiedBytes, err := patch.Apply([]byte(cache))
		expect.Nil(err)
		cache = string(modifiedBytes)
	}

	var object objects.Object
	err = json.Unmarshal([]byte(cache), &object)
	expect.Nil(err)

	aux, err := base64.StdEncoding.DecodeString(object.Data)
	expect.Nil(err)
	err = json.Unmarshal(aux, &settings)
	expect.Nil(err)

	return settings, cache
}

func getThings(server *katamari.Server) ([]objects.Object, error) {
	var objs []objects.Object
	if !server.Storage.Active() {
		err := server.Storage.Start([]string{}, nil)
		defer server.Storage.Close()
		if err != nil {
			return objs, err
		}
	}

	data, err := server.Storage.Get("things/*")
	if err != nil {
		return objs, err
	}

	objs, err = objects.DecodeList(data)
	if err != nil {
		return objs, err
	}

	return objs, nil
}

func makeGetNodes(server *katamari.Server) func() []string {
	return func() []string {
		var result []string
		thingsObjs, err := getThings(server)
		if err != nil {
			log.Println("failed to get things", err)
			return result
		}
		for _, thingObj := range thingsObjs {
			var thing Thing
			err = json.Unmarshal([]byte(thingObj.Data), &thing)
			if err == nil {
				result = append(result, thing.IP)
			}
		}

		return result
	}
}

func FakeServer(t *testing.T, pivotIP string) *katamari.Server {
	server := &katamari.Server{}
	server.Silence = true
	server.Static = true
	server.Storage = &katamari.MemoryStorage{}
	authStore := &katamari.MemoryStorage{}
	err := authStore.Start([]string{}, nil)
	require.NoError(t, err)
	go katamari.WatchStorageNoop(authStore)
	auth := auth.New(
		auth.NewJwtStore("key", time.Minute*10),
		authStore,
	)

	// Server routing
	server.Audit = func(r *http.Request) bool {
		// role, _, _ := auth.Audit(r)
		// if role == "root" {
		// 	return true
		// }

		// // default to unauthorized
		// return false
		return true
	}

	server.Client = &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			MaxConnsPerHost:   3000,
			DisableKeepAlives: true,
		},
	}
	server.Router = mux.NewRouter()
	getNodes := makeGetNodes(server)
	keys := []string{"things/*", "settings"}
	server.WriteFilter("things/*", katamari.NoopFilter)
	server.AfterFilter("things/*", pivot.SyncWriteFilter(server.Client, pivotIP, getNodes))
	server.ReadFilter("things/*", pivot.SyncReadFilter(server.Client, server.Storage, pivotIP, keys))
	server.DeleteFilter("things/*", pivot.SyncDeleteFilter(server.Client, pivotIP, server.Storage, "things", getNodes))
	server.WriteFilter("settings", katamari.NoopFilter)
	server.AfterFilter("settings", pivot.SyncWriteFilter(server.Client, pivotIP, getNodes))
	server.ReadFilter("settings", pivot.SyncReadFilter(server.Client, server.Storage, pivotIP, keys))
	pivot.Router(server.Router, server.Storage, server.Client, pivotIP, keys)
	auth.Router(server, pivotIP)
	server.Start("localhost:0")
	return server
}

func TestBasicPivotSync(t *testing.T) {
	var wg sync.WaitGroup
	var pivotThings []Thing
	var pivotThingsCache string
	var nodeThings []Thing
	var nodeThingsCache string
	var nodeSettings Settings
	var nodeSettingsCache string
	var pivotSettings Settings
	var pivotSettingsCache string

	pivotServer := FakeServer(t, "")
	defer pivotServer.Close(os.Interrupt)

	nodeServer := FakeServer(t, pivotServer.Address)
	defer nodeServer.Close(os.Interrupt)

	token := Register(t, pivotServer)
	require.NotEqual(t, "", token)
	authHeader := http.Header{}
	authHeader.Set("Authorization", "Bearer "+token)

	wsPivotThingsURL := url.URL{Scheme: "ws", Host: pivotServer.Address, Path: "/things/*"}
	wsPivotThingsClient, _, err := websocket.DefaultDialer.Dial(wsPivotThingsURL.String(), authHeader)
	require.NoError(t, err)

	wsNodeThingsURL := url.URL{Scheme: "ws", Host: nodeServer.Address, Path: "/things/*"}
	wsNodeThingsClient, _, err := websocket.DefaultDialer.Dial(wsNodeThingsURL.String(), authHeader)
	require.NoError(t, err)

	wsNodeSettingsURL := url.URL{Scheme: "ws", Host: nodeServer.Address, Path: "/settings"}
	wsNodeSettingsClient, _, err := websocket.DefaultDialer.Dial(wsNodeSettingsURL.String(), authHeader)
	require.NoError(t, err)

	wsPivotSettingsURL := url.URL{Scheme: "ws", Host: pivotServer.Address, Path: "/settings"}
	wsPivotSettingsClient, _, err := websocket.DefaultDialer.Dial(wsPivotSettingsURL.String(), authHeader)
	require.NoError(t, err)

	wg.Add(4)
	go func() {
		for {
			_, message, err := wsPivotThingsClient.ReadMessage()
			require.NoError(t, err)
			pivotThings, pivotThingsCache = decodeThingsData(message, pivotThingsCache)
			wg.Done()
		}
	}()

	go func() {
		for {
			_, message, err := wsNodeThingsClient.ReadMessage()
			require.NoError(t, err)
			nodeThings, nodeThingsCache = decodeThingsData(message, nodeThingsCache)
			wg.Done()
		}
	}()

	go func() {
		for {
			_, message, err := wsNodeSettingsClient.ReadMessage()
			require.NoError(t, err)
			nodeSettings, nodeSettingsCache = decodeSettingsData(message, nodeSettingsCache)
			wg.Done()
		}
	}()

	go func() {
		for {
			_, message, err := wsPivotSettingsClient.ReadMessage()
			require.NoError(t, err)
			pivotSettings, pivotSettingsCache = decodeSettingsData(message, pivotSettingsCache)
			wg.Done()
		}
	}()
	wg.Wait()

	require.Equal(t, 0, nodeSettings.DayEpoch)
	require.Equal(t, 0, pivotSettings.DayEpoch)

	wg.Add(2)
	thingID := CreateThing(t, pivotServer, token, nodeServer.Address)
	wg.Wait()

	require.Equal(t, 1, len(pivotThings))
	require.Equal(t, 1, len(nodeThings))
	require.Equal(t, false, pivotThings[0].On)
	require.Equal(t, false, nodeThings[0].On)
	wg.Wait()
	wg.Add(2)
	thingData := ReadThing(t, pivotServer, token, thingID)
	ModifyThing(t, pivotServer, token, thingData, thingID, true)

	wg.Wait()
	pivotThingData := ReadThing(t, pivotServer, token, thingID)
	nodeThingData := ReadThing(t, nodeServer, token, thingID)
	require.Equal(t, true, bytes.Equal(pivotThingData, nodeThingData))
	require.Equal(t, true, pivotThings[0].On)
	require.Equal(t, true, nodeThings[0].On)
	wg.Add(2)
	ModifyThing(t, pivotServer, token, thingData, thingID, false)

	wg.Wait()
	require.Equal(t, false, pivotThings[0].On)
	require.Equal(t, false, nodeThings[0].On)
	wg.Add(2)
	ModifyThing(t, pivotServer, token, thingData, thingID, true)

	wg.Wait()
	require.Equal(t, true, pivotThings[0].On)
	require.Equal(t, true, nodeThings[0].On)
	wg.Add(2)
	ModifyThing(t, pivotServer, token, thingData, thingID, false)

	wg.Wait()
	require.Equal(t, false, pivotThings[0].On)
	require.Equal(t, false, nodeThings[0].On)

	wg.Add(2)
	UpdateSettings(t, nodeServer, token, 1)
	wg.Wait()
	require.Equal(t, 1, nodeSettings.DayEpoch)
	require.Equal(t, 1, pivotSettings.DayEpoch)

	wg.Add(2)
	UpdateSettings(t, pivotServer, token, 9)
	wg.Wait()
	require.Equal(t, 9, nodeSettings.DayEpoch)
	require.Equal(t, 9, pivotSettings.DayEpoch)
}
