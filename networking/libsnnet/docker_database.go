//
// Copyright (c) 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package libsnnet

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/boltdb/bolt"
	"github.com/golang/glog"
)

//DbTable interface that needs to be supported
//for the table to be handled by the database
type DbTable interface {
	// Creates the backing map
	NewTable()

	// Name of the table as stored in the database
	Name() string

	// Allocates and returns a single value in the table
	NewElement() interface{}

	// Add an value to the in memory table
	Add(k string, v interface{}) error
}

// A TableDBProvider represents a persistent data base provider
// that can be used to store map, arrays or any other type of
// tables into a database
type TableDBProvider interface {
	//Initializes the Database
	DbInit(dir string, file string) error
	//Populates the DockerPlugin cache from the database
	DbTableRebuild(table DbTable) error
	//Closes the database
	DbClose() error
	//Creates the tables if the tables do not already exist in the database
	DbTableInit(tables []string) error
	//Adds the key value pair to the table
	DbAdd(table string, key string, value interface{}) error
	//Adds the key value pair to the table
	DbDelete(table string, key string) error
	//Retrives the value corresponding to the key from the table
	DbGet(table string, key string) (interface{}, error)
}

type tableBoltDB struct {
	Name string
	DB   *bolt.DB
}

func newTableBoltDb() *tableBoltDB {
	return &tableBoltDB{
		Name: "tableBolt.DB",
	}
}

type dbProvider tableBoltDB

//NewTableBoltDBProvider returns a bolt based database that conforms
//to the tableDBProvider interface
func NewTableBoltDBProvider() TableDBProvider {
	return (*dbProvider)(newTableBoltDb())
}

func (db *dbProvider) DbInit(dbDir string, dbFile string) error {

	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("Unable to create db directory (%s) %v", dbDir, err)
	}

	dbPath := path.Join(dbDir, dbFile)

	options := bolt.Options{
		Timeout: 3 * time.Second,
	}

	var err error
	db.DB, err = bolt.Open(dbPath, 0644, &options)
	if err != nil {
		return fmt.Errorf("initDb failed %v", err)
	}

	return err
}

func (db *dbProvider) DbClose() error {
	return db.DB.Close()
}

func (db *dbProvider) DbTableRebuild(table DbTable) error {
	tables := []string{table.Name()}
	if err := db.DbTableInit(tables); err != nil {
		return fmt.Errorf("dbInit failed %v", err)
	}

	table.NewTable()

	err := db.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(table.Name()))

		err := b.ForEach(func(k, v []byte) error {
			vr := bytes.NewReader(v)

			val := table.NewElement()
			if err := gob.NewDecoder(vr).Decode(val); err != nil {
				return fmt.Errorf("Decode Error: %v %v %v", string(k), string(v), err)
			}
			glog.Infof("%v key=%v, value=%v\n", table, string(k), val)

			return table.Add(string(k), val)
		})
		return err
	})
	return err
}

func (db *dbProvider) DbTableInit(tables []string) (err error) {

	glog.Infof("dbInit Tables := %v", tables)
	for i, v := range tables {
		glog.Infof("table[%v] := %v, %v", i, v, []byte(v))
	}

	err = db.DB.Update(func(tx *bolt.Tx) error {
		for _, table := range tables {
			_, err := tx.CreateBucketIfNotExists([]byte(table))
			if err != nil {
				return fmt.Errorf("Bucket creation error: %v %v", table, err)
			}
		}
		return nil
	})

	if err != nil {
		glog.Errorf("Table creation error %v", err)
	}

	return err
}

func (db *dbProvider) DbAdd(table string, key string, value interface{}) (err error) {

	err = db.DB.Update(func(tx *bolt.Tx) error {
		var v bytes.Buffer

		if err := gob.NewEncoder(&v).Encode(value); err != nil {
			glog.Errorf("Encode Error: %v %v", err, value)
			return err
		}

		bucket := tx.Bucket([]byte(table))
		if bucket == nil {
			return fmt.Errorf("Bucket %v not found", table)
		}

		err = bucket.Put([]byte(key), v.Bytes())
		if err != nil {
			return fmt.Errorf("Key Store error: %v %v %v %v", table, key, value, err)
		}
		return nil
	})

	return err
}

func (db *dbProvider) DbDelete(table string, key string) (err error) {

	err = db.DB.Update(func(tx *bolt.Tx) error {

		bucket := tx.Bucket([]byte(table))
		if bucket == nil {
			return fmt.Errorf("Bucket %v not found", table)
		}

		err = bucket.Delete([]byte(key))
		if err != nil {
			return fmt.Errorf("Key Delete error: %v %v ", key, err)
		}
		return nil
	})

	return err
}

func (db *dbProvider) DbGet(table string, key string) (value interface{}, err error) {

	err = db.DB.View(func(tx *bolt.Tx) error {

		bucket := tx.Bucket([]byte(table))
		if bucket == nil {
			return fmt.Errorf("Bucket %v not found", table)
		}

		val := bucket.Get([]byte(key))
		if val == nil {
			return nil
		}

		v := bytes.NewReader(val)
		if err := gob.NewDecoder(v).Decode(value); err != nil {
			glog.Errorf("Decode Error: %v %v %v", table, key, err)
			return err
		}

		return nil
	})

	return value, err
}
