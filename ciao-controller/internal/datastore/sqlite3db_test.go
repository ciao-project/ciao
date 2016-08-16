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

package datastore

import (
	"testing"
	"time"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/ciao-storage"
	"github.com/01org/ciao/ssntp/uuid"
)

func TestGetWorkloadStorage(t *testing.T) {
	config := Config{
		PersistentURI: "file:memdb3?mode=memory&cache=shared",
		TransientURI:  "file:memdb4?mode=memory&cache=shared",
	}

	db, err := getPersistentStore(config)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.getWorkloadStorage("validid")
	if err != nil {
		t.Fatal(err)
	}

	db.disconnect()
}

func TestGetTenantDevices(t *testing.T) {
	config := Config{
		PersistentURI: "file:memdb5?mode=memory&cache=shared",
		TransientURI:  "file:memdb6?mode=memory&cache=shared",
	}

	db, err := getPersistentStore(config)
	if err != nil {
		t.Fatal(err)
	}

	blockDevice := storage.BlockDevice{
		ID: uuid.Generate().String(),
	}

	data := types.BlockData{
		BlockDevice: blockDevice,
		Size:        0,
		State:       types.Available,
		TenantID:    uuid.Generate().String(),
		CreateTime:  time.Now(),
	}

	err = db.createBlockData(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure our query works.
	devices, err := db.getTenantDevices(data.TenantID)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := devices[data.ID]
	if !ok {
		t.Fatal(err)
	}

	db.disconnect()
}

func TestGetAllBlockData(t *testing.T) {
	config := Config{
		PersistentURI: "file:memdb7?mode=memory&cache=shared",
		TransientURI:  "file:memdb8?mode=memory&cache=shared",
	}

	db, err := getPersistentStore(config)
	if err != nil {
		t.Fatal(err)
	}

	blockDevice := storage.BlockDevice{
		ID: uuid.Generate().String(),
	}

	data := types.BlockData{
		BlockDevice: blockDevice,
		Size:        0,
		State:       types.Available,
		TenantID:    uuid.Generate().String(),
		CreateTime:  time.Now(),
	}

	err = db.createBlockData(data)
	if err != nil {
		t.Fatal(err)
	}

	devices, err := db.getAllBlockData()
	if err != nil {
		t.Fatal(err)
	}

	_, ok := devices[data.ID]
	if !ok {
		t.Fatal(err)
	}

	db.disconnect()
}
