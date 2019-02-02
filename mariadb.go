package samo

import (
	"database/sql"
	"errors"
	"time"

	_ "github.com/go-sql-driver/mysql" // idk
)

// MariaDbStorage : composition of storage
type MariaDbStorage struct {
	User     string
	Password string
	Name     string
	mysql    *sql.DB
	*Storage
}

// Active  :
func (db *MariaDbStorage) Active() bool {
	return db.Storage.Active
}

// Start  :
func (db *MariaDbStorage) Start(separator string) error {
	var err error
	db.Storage.Separator = separator
	db.mysql, err = sql.Open("mysql", db.User+":"+db.Password+"@/"+db.Name)
	if err == nil {
		db.Storage.Active = true
	}
	return err
}

// Close  :
func (db *MariaDbStorage) Close() {
	if db.mysql != nil {
		db.Storage.Active = false
		db.mysql.Close()
	}
}

// Keys  :
func (db *MariaDbStorage) Keys() ([]byte, error) {
	stats := Stats{
		Keys: []string{},
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
	newObject.Data = Data
	newObject.Index = Index
	return newObject, nil
}

// Get  :
func (db *MariaDbStorage) Get(mode string, key string) ([]byte, error) {
	if mode == "sa" {
		rows, err := db.mysql.Query("call `getSa`('" + key + "');")
		if err != nil {
			return []byte(""), err
		}
		defer rows.Close()
		var newObject Object
		empty := true
		for rows.Next() {
			newObject, err = db.fromRowToObj(rows)
			if err != nil {
				return nil, err
			}
			empty = false
		}

		if empty {
			return []byte(""), errors.New("samo: not found")
		}

		return db.Storage.Objects.encode(newObject)
	}

	if mode == "mo" {
		res := []Object{}
		rows, err := db.mysql.Query("call `getMo`('" + key + "');")
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
	_, err := db.mysql.Query("call `set`('" + key + "', '" + data + "');")
	if err != nil {
		return "", err
	}

	return index, nil
}

// Del  :
func (db *MariaDbStorage) Del(key string) error {
	rows, err := db.mysql.Query("call `getSa`('" + key + "');")
	defer rows.Close()
	empty := true
	for rows.Next() {
		empty = false
	}
	if empty {
		return errors.New("samo: not found")
	}
	_, err = db.mysql.Query("call `del`('" + key + "');")
	if err != nil {
		return err
	}
	return nil
}
