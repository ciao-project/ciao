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
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/01org/ciao/database"
)

// Boltdb implements the DataStore interface for persistent data
// on a key/value database with BoltDB
type Boltdb struct {
	database.DbProvider
	DbDir  string
	DbFile string
}

// Write is the boltdb image metadata write implementation.
func (b *Boltdb) Write(i Image) error {
	b.DbInit(b.DbDir, b.DbFile)
	defer b.DbClose()

	err := b.DbAdd("images", i.ID, i)

	if err != nil {
		return err
	}

	return nil
}

// Delete is the boltdb image metadata delete implementation.
func (b *Boltdb) Delete(id string) error {
	b.DbInit(b.DbDir, b.DbFile)
	defer b.DbClose()
	return b.DbDelete("images", id)
}

// Get is the boltdb image metadata get implementation.
func (b *Boltdb) Get(ID string) (Image, error) {
	b.DbInit(b.DbDir, b.DbFile)
	defer b.DbClose()

	img := Image{}
	data, err := b.DbGet("images", ID)
	if err != nil {
		return img, fmt.Errorf("Error on image retrieve: %v ", err)
	}
	vr := bytes.NewReader(data.([]byte))
	if err := gob.NewDecoder(vr).Decode(&img); err != nil {
		return img, fmt.Errorf("Decode Error: %v", err)
	}

	return img, err
}

// GetAll is the boltdb image metadata get all images implementation.
func (b *Boltdb) GetAll() (images []Image, err error) {
	b.DbInit(b.DbDir, b.DbFile)
	defer b.DbClose()
	var elements []interface{}

	elements, err = b.DbProvider.DbGetAll("images")

	for _, data := range elements {
		if data != nil {
			img := Image{}
			vr := bytes.NewReader(data.([]byte))
			if err := gob.NewDecoder(vr).Decode(&img); err != nil {
				return images, fmt.Errorf("Decode Error: %v", err)
			}
			images = append(images, img)
		}
	}

	return images, err
}
