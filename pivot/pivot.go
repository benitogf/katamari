package pivot

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/benitogf/katamari"
	"github.com/benitogf/katamari/key"
	"github.com/benitogf/katamari/objects"
	"github.com/gorilla/mux"
)

// ActivityEntry keeps the time of the last entry
type ActivityEntry struct {
	LastEntry int64 `json:"lastEntry"`
}

// GetNodes function that returns the nodes ips
type GetNodes func() []string

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func lastActivity(objs []objects.Object) int64 {
	if len(objs) > 0 {
		return max(objs[0].Created, objs[0].Updated)
	}

	return 0
}

func checkLastDelete(storage katamari.Database, lastEntry int64, key string) int64 {
	lastDelete, err := storage.Get("pivot:" + key)
	if err != nil {
		// log.Println("failed to get last delete of ", key, err)
		return lastEntry
	}

	obj, err := objects.DecodeRaw(lastDelete)
	if err != nil {
		// log.Println("failed to decode object of last delete for ", key, lastDelete, err)
		return lastEntry
	}

	lastDeleteNum, err := strconv.Atoi(obj.Data)
	if err != nil {
		// log.Println("failed to decode last delete of ", key, lastDelete, err)
		return lastEntry
	}

	return max(lastEntry, int64(lastDeleteNum))
}

func checkActivity(storage katamari.Database, _key string) (ActivityEntry, error) {
	var activity ActivityEntry
	entries, err := storage.Get(_key)
	if err != nil {
		// log.Println("failed to fetch local "+_key, err)
		return activity, nil
	}
	baseKey := _key

	if key.LastIndex(_key) == "*" {
		baseKey = strings.Replace(_key, "/*", "", 1)
		objs, err := objects.DecodeListRaw(entries)
		if _key != "users/*" {
			objs, err = objects.DecodeListData(objs)
		}
		if err != nil {
			// log.Println("failed to decode "+_key+" objects list", err)
			return activity, err
		}

		activity.LastEntry = checkLastDelete(storage, lastActivity(objs), baseKey)
		return activity, nil
	}

	obj, err := objects.DecodeRaw(entries)
	if err != nil {
		// log.Println("failed to decode "+_key+" objects list", err)
		return activity, err
	}

	activity.LastEntry = checkLastDelete(storage, max(obj.Created, obj.Updated), baseKey)
	return activity, nil
}

func checkPivotActivity(client *http.Client, pivot string, key string) (ActivityEntry, error) {
	var activity ActivityEntry
	resp, err := client.Get("http://" + pivot + "/activity/" + key)
	if err != nil {
		// log.Println("failed to get activity on "+key+" from pivot at "+pivot, err)
		return activity, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return activity, errors.New("failed to get activity on " + key + " from pivot at " + pivot)
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&activity)

	return activity, err
}

func getEntriesFromPivot(client *http.Client, pivot string, key string) ([]objects.Object, error) {
	var objs []objects.Object
	resp, err := client.Get("http://" + pivot + "/" + key)
	if err != nil {
		// log.Println("failed to get "+key+" from pivot", err)
		return objs, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return objs, errors.New("failed to get " + key + " from pivot " + resp.Status)
	}

	objs, err = objects.DecodeListFromReader(resp.Body)
	if err != nil {
		return objs, err
	}

	if key != "users/*" {
		return objects.DecodeListData(objs)
	}

	return objs, nil
}

// TriggerNodeSync will call pivot on a node server
func TriggerNodeSync(client *http.Client, node string) {
	// log.Println("node sync", node)
	resp, err := client.Get("http://" + node + "/pivot")
	if err != nil {
		// log.Println("failed to trigger sync from pivot on ", node, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// log.Println("failed to trigger sync from pivot on " + node + " " + resp.Status)
		return
	}
}

func getEntryFromPivot(client *http.Client, pivot string, key string) (objects.Object, error) {
	var obj objects.Object
	resp, err := client.Get("http://" + pivot + "/" + key)
	if err != nil {
		// log.Println("failed to get "+key+" from pivot", err)
		return obj, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return obj, errors.New("failed to get " + key + " from pivot " + resp.Status)
	}

	return objects.DecodeFromReader(resp.Body)
}

// get from pivot and write to local
func syncLocalEntries(client *http.Client, storage katamari.Database, pivot string, _key string, lastEntry int64) error {
	if key.LastIndex(_key) == "*" {
		baseKey := strings.Replace(_key, "/*", "", 1)
		objsPivot, err := getEntriesFromPivot(client, pivot, _key)
		if err != nil {
			// log.Println("sync local " + baseKey + " failed to get from pivot")
			return err
		}

		localData, err := storage.Get(_key)
		if err != nil {
			// log.Println("sync local " + _key + " failed to read local entries")
			return err
		}

		objsLocal, err := objects.DecodeListRaw(localData)
		if _key != "users/*" {
			objsLocal, err = objects.DecodeListData(objsLocal)
		}
		if err != nil {
			// log.Println("sync local " + _key + " failed to decode local entries")
			return err
		}

		objsToDelete := getEntriesNegativeDiff(objsLocal, objsPivot)
		for _, index := range objsToDelete {
			storage.Del(baseKey + "/" + index)
		}

		objsToSend := getEntriesPositiveDiff(objsLocal, objsPivot)
		for _, obj := range objsToSend {
			if _key != "users/*" {
				obj.Data = base64.StdEncoding.EncodeToString([]byte(obj.Data))
			}
			storage.Pivot(baseKey+"/"+obj.Index, obj.Data, obj.Created, obj.Updated)
			// if err != nil {
			// 	log.Println("failed to store entry from pivot", err)
			// }
		}
		storage.Set("pivot:"+baseKey, strconv.FormatInt(lastEntry, 10))
		return nil
	}

	obj, err := getEntryFromPivot(client, pivot, _key)
	if err != nil {
		// log.Println("sync local " + _key + " failed to get from pivot")
		return err
	}
	storage.Pivot(_key, obj.Data, obj.Created, obj.Updated)
	// if err != nil {
	// 	log.Println("failed to store entry from pivot", err)
	// }
	storage.Set("pivot:"+_key, strconv.FormatInt(lastEntry, 10))

	return nil
}

func sendToPivot(client *http.Client, key string, pivot string, obj objects.Object) error {
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(obj)
	resp, err := client.Post("http://"+pivot+"/pivot/"+key, "application/json", buf)
	if err != nil {
		// log.Println("failed to send update to pivot", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// log.Println("http://" + pivot + "/pivot/" + key)
		// log.Println("failed to send update to pivot " + resp.Status)
		return errors.New("failed to send update to pivot " + resp.Status)
	}

	return nil
}

func sendDelete(client *http.Client, key, pivot string, lastEntry int64) error {
	req, err := http.NewRequest("DELETE", "http://"+pivot+"/pivot/"+key+"/"+strconv.FormatInt(lastEntry, 10), nil)
	if err != nil {
		// log.Println("failed to send delete to pivot", err)
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		// log.Println("failed to send delete to pivot", err)
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		// log.Println("http://" + pivot + "/pivot/" + key)
		// log.Println("failed to send delete to pivot " + resp.Status)
		return errors.New("failed to send delete to pivot " + resp.Status)
	}

	return nil
}

func getEntriesNegativeDiff(objsDst, objsSrc []objects.Object) []string {
	var result []string
	for _, objDst := range objsDst {
		found := false
		for _, objSrc := range objsSrc {
			if objSrc.Index == objDst.Index {
				found = true
				break
			}
		}
		if !found {
			result = append(result, objDst.Index)
		}
	}
	return result
}

func getEntriesPositiveDiff(objsDst, objsSrc []objects.Object) []objects.Object {
	var result []objects.Object
	for _, objSrc := range objsSrc {
		needsUpdate := false
		found := false
		for _, objDst := range objsDst {
			if objSrc.Index == objDst.Index {
				found = true
			}
			if objSrc.Index == objDst.Index && objSrc.Updated > objDst.Updated {
				needsUpdate = true
				break
			}
		}
		if needsUpdate || !found {
			result = append(result, objSrc)
		}
	}
	return result
}

// get from local and send to pivot (updates only, no new entries or deletes)
func syncPivotEntries(client *http.Client, storage katamari.Database, pivot string, _key string, lastEntry int64) error {
	localData, err := storage.Get(_key)
	if err != nil {
		// log.Println("sync pivot " + _key + " failed to read local entries")
		return err
	}
	if key.LastIndex(_key) == "*" {
		baseKey := strings.Replace(_key, "/*", "", 1)
		objsLocal, err := objects.DecodeListRaw(localData)
		if _key != "users/*" {
			objsLocal, err = objects.DecodeListData(objsLocal)
		}
		if err != nil {
			// log.Println("sync pivot " + _key + " failed to decode local entries")
			return err
		}

		objsPivot, err := getEntriesFromPivot(client, pivot, _key)
		if err != nil {
			// log.Println("sync pivot " + baseKey + " failed to get from pivot")
			return err
		}

		objsToDelete := getEntriesNegativeDiff(objsPivot, objsLocal)
		for _, index := range objsToDelete {
			sendDelete(client, baseKey+"/"+index, pivot, lastEntry)
		}

		objsToSend := getEntriesPositiveDiff(objsPivot, objsLocal)
		for _, obj := range objsToSend {
			if _key != "users/*" {
				obj.Data = base64.StdEncoding.EncodeToString([]byte(obj.Data))
			}
			sendToPivot(client, baseKey+"/"+obj.Index, pivot, obj)
		}

		return nil
	}

	obj, err := objects.DecodeRaw(localData)
	if err != nil {
		// log.Println("sync pivot " + _key + " failed to decode local entries")
		return err
	}
	sendToPivot(client, obj.Index, pivot, obj)

	return nil
}

func synchronizeItem(client *http.Client, storage katamari.Database, pivot string, key string) error {
	update := false
	_key := strings.Replace(key, "/*", "", 1)
	//check
	activityPivot, err := checkPivotActivity(client, pivot, _key)
	if err != nil {
		// log.Println(err)
		return errors.New("failed to check activity for " + _key + " on pivot")
	}
	activityLocal, err := checkActivity(storage, key)
	if err != nil {
		return errors.New("failed to check activity for " + _key + " on local")
	}

	// sync
	if activityLocal.LastEntry > activityPivot.LastEntry {
		err := syncPivotEntries(client, storage, pivot, key, activityLocal.LastEntry)
		if err != nil {
			return err
		}
		update = true
	}

	if activityLocal.LastEntry < activityPivot.LastEntry {
		err := syncLocalEntries(client, storage, pivot, key, activityPivot.LastEntry)
		if err != nil {
			return err
		}
		update = true
	}

	if update {
		return nil
	}

	return errors.New("nothing to synchronize for " + key)
}

// Synchronize a list of keys
func Synchronize(client *http.Client, storage katamari.Database, pivot string, keys []string) error {
	update := false
	for _, key := range keys {
		errItem := synchronizeItem(client, storage, pivot, key)
		if errItem == nil {
			update = true
		}
	}
	if update {
		return nil
	}
	return errors.New("nothing to synchronize")
}

// SyncReadFilter filter to synchronize with pivot on read
func SyncReadFilter(client *http.Client, storage katamari.Database, pivot string, keys []string) katamari.Apply {
	return func(index string, data []byte) ([]byte, error) {
		if pivot != "" {
			// log.Println("read filter", index)
			err := Synchronize(client, storage, pivot, keys)
			if err != nil {
				return data, nil
			}

			updatedData, err := storage.Get(index)
			if err != nil {
				return data, nil
			}

			return updatedData, nil
		}

		return data, nil
	}
}

// SyncWriteFilter filter to synchronize nodes on write
func SyncWriteFilter(client *http.Client, pivotIP string, getNodes GetNodes) katamari.Notify {
	return func(index string) {
		// log.Println("sync write", index)
		if pivotIP == "" {
			for _, node := range getNodes() {
				go TriggerNodeSync(client, node)
			}
		}
	}
}

// SyncDeleteFilter update the last delete time on each delete
func SyncDeleteFilter(client *http.Client, pivotIP string, storage katamari.Database, key string, getNodes GetNodes) katamari.ApplyDelete {
	return func(index string) error {
		if pivotIP == "" {
			for _, node := range getNodes() {
				go TriggerNodeSync(client, node)
			}
		}

		storage.Set("pivot:"+key, katamari.Time())
		return nil
	}
}

// Pivot endpoint to trigger a synchronize from the pivot server
func Pivot(client *http.Client, storage katamari.Database, pivot string, keys []string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if pivot == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "this method should not be called on the pivot server")
			return
		}

		Synchronize(client, storage, pivot, keys)
		w.WriteHeader(http.StatusOK)
	}
}

// Get WILL
func Get(storage katamari.Database, key string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var objs []objects.Object
		raw, err := storage.Get(key)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			return
		}

		err = json.Unmarshal(raw, &objs)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(objs)
	}
}

// Set set data on the pivot instance
func Set(storage katamari.Database, key string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		decoded, err := objects.DecodeFromReader(r.Body)
		if err != nil {
			// log.Println("failed to decode "+key+" entry on pivot", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		index := mux.Vars(r)["index"]
		itemKey := key + "/" + decoded.Index
		if index == "" {
			itemKey = key
		}
		_, err = storage.Pivot(itemKey, decoded.Data, decoded.Created, decoded.Updated)
		if err != nil {
			// log.Println("failed to store on pivot "+key+" entry", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// Delete delete data on the pivot instance
func Delete(storage katamari.Database, key string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		index := mux.Vars(r)["index"]
		time := mux.Vars(r)["time"]
		itemKey := key + "/" + index
		err := storage.Del(itemKey)
		storage.Set("pivot:"+key, time)
		if err != nil {
			// log.Println("failed to delete on pivot "+key+" entry", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// Activity route to get activity info from the pivot instance
func Activity(storage katamari.Database, key string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if storage != nil && storage.Active() {
			activity, _ := checkActivity(storage, key)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(activity)
		}
	}
}

// Router will add the router needed to synchronize the keys
func Router(router *mux.Router, storage katamari.Database, client *http.Client, pivot string, keys []string) {
	router.HandleFunc("/pivot", Pivot(client, storage, pivot, keys)).Methods("GET")
	for _, key := range keys {
		baseKey := strings.Replace(key, "/*", "", 1)
		if key == "users/*" {
			router.HandleFunc("/users/*", Get(storage, key)).Methods("GET")
		}
		router.HandleFunc("/activity/"+baseKey, Activity(storage, key)).Methods("GET")
		if baseKey != key {
			router.HandleFunc("/pivot/"+baseKey+"/{index:[a-zA-Z\\*\\d\\/]+}", Set(storage, baseKey)).Methods("POST")
			router.HandleFunc("/pivot/"+baseKey+"/{index:[a-zA-Z\\*\\d\\/]+}/{time:[a-zA-Z\\*\\d\\/]+}", Delete(storage, baseKey)).Methods("DELETE")
		} else {
			router.HandleFunc("/pivot/"+baseKey, Set(storage, baseKey)).Methods("POST")
			router.HandleFunc("/pivot/"+baseKey+"/{time:[a-zA-Z\\*\\d\\/]+}", Delete(storage, baseKey)).Methods("DELETE")
		}
	}
}
