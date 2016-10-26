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

package database

import (
	"os"
	"path"
	"sync"
	"testing"
)

const (
	tableTestMap = "tests"
)

type Provider struct {
	Db       DbProvider
	DbTables []string
	DbDir    string
	DbFile   string
}

type TestData struct {
	ID string
}

// TestMap stores Image metadata
type TestMap struct {
	sync.RWMutex
	m map[string]*TestData
}

// NewTable creates a new map of Images
func (i *TestMap) NewTable() {
	i.m = make(map[string]*TestData)
}

// Name provides Images table name
func (i *TestMap) Name() string {
	return tableTestMap
}

// NewElement returns a new Test struct
func (i *TestMap) NewElement() interface{} {
	return &TestData{}
}

var dbTables = []string{"tests"}
var dbDir = "/tmp"
var dbFile = "database.db"

//  closeDb is a generic function to close every Db transaction
func closeDb(provider *Provider) {
	_ = provider.Db.DbClose()
}

func initProvider(dbProvider DbProvider) Provider {
	provider := Provider{
		Db:       dbProvider,
		DbFile:   dbFile,
		DbDir:    dbDir,
		DbTables: dbTables,
	}
	return provider
}

func testDbInit(t *testing.T, provider Provider) {
	err := provider.Db.DbInit(provider.DbDir, provider.DbFile)
	if err != nil {
		t.Fatal(err)
	}
	defer closeDb(&provider)
}

func testDbClose(t *testing.T, provider Provider) {
	err := provider.Db.DbInit(provider.DbDir, provider.DbFile)
	if err != nil {
		t.Fatal(err)
	}
	err = provider.Db.DbClose()
	if err != nil {
		t.Fatal(err)
	}
}

func testDbTableInit(t *testing.T, provider Provider) {
	defer closeDb(&provider)

	err := provider.Db.DbInit(provider.DbDir, provider.DbFile)
	if err != nil {
		t.Fatal(err)
	}

	err = provider.Db.DbTablesInit(provider.DbTables)
	if err != nil {
		t.Fatal(err)
	}
}

func testDbAdd(t *testing.T, provider Provider) {
	defer closeDb(&provider)

	err := provider.Db.DbInit(provider.DbDir, provider.DbFile)
	if err != nil {
		t.Fatal(err)
	}

	err = provider.Db.DbTablesInit(provider.DbTables)
	if err != nil {
		t.Fatal(err)
	}

	err = provider.Db.DbAdd(provider.DbTables[0], "sampleKey", TestData{})
	if err != nil {
		t.Fatal(err)
	}
}

func testDbDelete(t *testing.T, provider Provider) {
	defer closeDb(&provider)

	err := provider.Db.DbInit(provider.DbDir, provider.DbFile)
	if err != nil {
		t.Fatal(err)
	}

	err = provider.Db.DbTablesInit(provider.DbTables)
	if err != nil {
		t.Fatal(err)
	}

	err = provider.Db.DbAdd(provider.DbTables[0], "sampleKey", TestData{ID: "sampleKey"})
	if err != nil {
		t.Fatal(err)
	}

	err = provider.Db.DbDelete(provider.DbTables[0], "sampleKey")
	if err != nil {
		t.Fatal(err)
	}
}

func testDbGet(t *testing.T, provider Provider) {
	defer closeDb(&provider)

	err := provider.Db.DbInit(provider.DbDir, provider.DbFile)
	if err != nil {
		t.Fatal(err)
	}

	err = provider.Db.DbTablesInit(provider.DbTables)
	if err != nil {
		t.Fatal(err)
	}

	err = provider.Db.DbAdd(provider.DbTables[0], "sampleKey", TestData{ID: "sampleKey"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = provider.Db.DbGet(provider.DbTables[0], "sampleKey", &TestMap{})
	if err != nil {
		t.Fatal(err)
	}
}

func testDbGetAll(t *testing.T, provider Provider) {
	defer closeDb(&provider)

	err := provider.Db.DbInit(provider.DbDir, provider.DbFile)
	if err != nil {
		t.Fatal(err)
	}

	err = provider.Db.DbTablesInit(provider.DbTables)
	if err != nil {
		t.Fatal(err)
	}

	err = provider.Db.DbAdd(provider.DbTables[0], "sampleKey", TestData{ID: "sampleKey"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = provider.Db.DbGet(provider.DbTables[0], "sampleKey", &TestMap{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = provider.Db.DbGetAll(provider.DbTables[0], &TestMap{})
	if err != nil {
		t.Fatal(err)
	}
}

// Test for BoltDb Provider

func TestBoltDbInit(t *testing.T) {
	provider := initProvider(NewBoltDBProvider())
	testDbInit(t, provider)
	_ = os.Remove(path.Join(dbDir, dbFile))
}

func TestBoltDbClose(t *testing.T) {
	provider := initProvider(NewBoltDBProvider())
	testDbClose(t, provider)
	_ = os.Remove(path.Join(dbDir, dbFile))
}

func TestBoltDbTableInit(t *testing.T) {
	provider := initProvider(NewBoltDBProvider())
	testDbTableInit(t, provider)
	_ = os.Remove(path.Join(dbDir, dbFile))
}

func TestBoltDbAdd(t *testing.T) {
	provider := initProvider(NewBoltDBProvider())
	testDbAdd(t, provider)
	_ = os.Remove(path.Join(dbDir, dbFile))
}

func TestBoltDbDelete(t *testing.T) {
	provider := initProvider(NewBoltDBProvider())
	testDbDelete(t, provider)
	_ = os.Remove(path.Join(dbDir, dbFile))
}

func TestBoltDbGet(t *testing.T) {
	provider := initProvider(NewBoltDBProvider())
	testDbGet(t, provider)
	_ = os.Remove(path.Join(dbDir, dbFile))
}

func TestBoltDbGetAll(t *testing.T) {
	provider := initProvider(NewBoltDBProvider())
	testDbGetAll(t, provider)
	_ = os.Remove(path.Join(dbDir, dbFile))
}
