package samo

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql" // idk
)

// MariaDbStorage : composition of storage
type MariaDbStorage struct {
	User     string
	Password string
	Name     string
	mysql    *sql.DB
	mutex    sync.RWMutex
	*Storage
}

func (db *MariaDbStorage) trimQuotes(s string) string {
	if len(s) > 0 && s[0] == '"' {
		s = s[1:]
	}
	if len(s) > 0 && s[len(s)-1] == '"' {
		s = s[:len(s)-1]
	}
	return s
}

// Active  :
func (db *MariaDbStorage) Active() bool {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	return db.Storage.Active
}

// Start  :
func (db *MariaDbStorage) Start() error {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	var err error
	if db.Storage.Separator == "" {
		db.Storage.Separator = "/"
	}
	db.mysql, err = sql.Open("mysql", db.User+":"+db.Password+"@/"+db.Name)
	if err != nil {
		return err
	}
	db.Storage.Active = true
	db.mysql.Close()
	return nil
}

// Close  :
func (db *MariaDbStorage) Close() {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	db.Storage.Active = false
}

// Clear  :
func (db *MariaDbStorage) Clear() {
	var err error
	db.mutex.Lock()
	defer db.mutex.Unlock()
	if !db.Storage.Active {
		db.mysql, err = sql.Open("mysql", db.User+":"+db.Password+"@/"+db.Name)
		defer db.mysql.Close()
		if err != nil {
			return
		}
		_, _ = db.mysql.Query("call `clear`();")
	}
}

// Keys  :
func (db *MariaDbStorage) Keys() ([]byte, error) {
	stats := Stats{
		Keys: []string{},
	}
	var err error
	db.mysql, err = sql.Open("mysql", db.User+":"+db.Password+"@/"+db.Name)
	defer db.mysql.Close()
	if err != nil {
		return nil, err
	}
	rows, err := db.mysql.Query("call `keys`();")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var k string
		err := rows.Scan(&k)
		if err != nil {
			return nil, err
		}
		stats.Keys = append(stats.Keys, k)
	}

	return db.Storage.Objects.encode(stats)
}

func (db *MariaDbStorage) fromRowToObj(rows *sql.Rows) (Object, error) {
	var newObject Object
	var (
		Created string
		Updated string
		Data    string
		Index   string
	)
	err := rows.Scan(&Index, &Data, &Created, &Updated)
	if err != nil {
		return newObject, err
	}
	nC, err := time.Parse("2006-01-02 15:04:05", Created)
	if err != nil {
		return newObject, err
	}
	if Updated != "0000-00-00 00:00:00" {
		nU, err := time.Parse("2006-01-02 15:04:05", Updated)
		if err != nil {
			return newObject, err
		}
		newObject.Updated = nU.UnixNano()
	} else {
		newObject.Updated = int64(0)
	}
	newObject.Created = nC.UnixNano()
	newObject.Data = fmt.Sprintf(`%s`, db.trimQuotes(Data))
	newObject.Index = Index
	return newObject, nil
}

// Get  :
func (db *MariaDbStorage) Get(mode string, key string) ([]byte, error) {
	var err error
	db.mysql, err = sql.Open("mysql", db.User+":"+db.Password+"@/"+db.Name)
	defer db.mysql.Close()
	if err != nil {
		return nil, err
	}
	if mode == "sa" {
		rows, err := db.mysql.Query("call `getSa`('" + key + "');")
		if err != nil {
			return []byte(""), err
		}
		defer rows.Close()
		var newObject Object
		empty := !rows.Next()

		if empty {
			return []byte(""), errors.New("samo: not found")
		}

		newObject, err = db.fromRowToObj(rows)
		if err != nil {
			return nil, err
		}

		return db.Storage.Objects.encode(newObject)
	}

	if mode == "mo" {
		res := []Object{}
		rows, err := db.mysql.Query("call `getMo`('" + key + "', '" + db.Storage.Separator + "');")
		if err != nil {
			return []byte(""), err
		}
		defer rows.Close()
		for rows.Next() {
			newObject, err := db.fromRowToObj(rows)
			if err == nil {
				res = append(res, newObject)
			}
		}

		return db.Storage.Objects.encode(res)
	}

	return []byte(""), errors.New("samo: unrecognized mode: " + mode)
}

// Set  :
func (db *MariaDbStorage) Set(key string, index string, now int64, data string) (string, error) {
	var err error
	db.mysql, err = sql.Open("mysql", db.User+":"+db.Password+"@/"+db.Name)
	defer db.mysql.Close()
	if err != nil {
		return "", err
	}
	_, err = db.mysql.Query("call `set`('" + key + "', '" + fmt.Sprintf(`%#v`, data) + "');")
	if err != nil {
		return "", err
	}

	return index, nil
}

// Del  :
func (db *MariaDbStorage) Del(key string) error {
	var err error
	db.mysql, err = sql.Open("mysql", db.User+":"+db.Password+"@/"+db.Name)
	defer db.mysql.Close()
	if err != nil {
		return err
	}
	rows, err := db.mysql.Query("call `getSa`('" + key + "');")
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return errors.New("samo: not found")
	}

	_, err = db.mysql.Query("call `del`('" + key + "');")
	if err != nil {
		return err
	}
	return nil
}
