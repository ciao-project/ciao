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

// MetaDs implements the DataStore interface for persistent data
type MetaDs struct {
	database.DbProvider
	DbDir  string
	DbFile string
}

// Write is the metadata write implementation.
func (m *MetaDs) Write(i Image) error {
	m.DbInit(m.DbDir, m.DbFile)
	defer m.DbClose()

	err := m.DbAdd("images", i.ID, i)

	if err != nil {
		return err
	}

	return nil
}

// Delete is the metadata delete implementation.
func (m *MetaDs) Delete(id string) error {
	m.DbInit(m.DbDir, m.DbFile)
	defer m.DbClose()
	return m.DbDelete("images", id)
}

// Get is the metadata get implementation.
func (m *MetaDs) Get(ID string) (Image, error) {
	m.DbInit(m.DbDir, m.DbFile)
	defer m.DbClose()

	img := Image{}
	data, err := m.DbGet("images", ID)
	if err != nil {
		return img, fmt.Errorf("Error on image retrieve: %v ", err)
	}
	vr := bytes.NewReader(data.([]byte))
	if err := gob.NewDecoder(vr).Decode(&img); err != nil {
		return img, fmt.Errorf("Decode Error: %v", err)
	}

	return img, err
}

// GetAll is the metadata get all images implementation.
func (m *MetaDs) GetAll() (images []Image, err error) {
	m.DbInit(m.DbDir, m.DbFile)
	defer m.DbClose()
	var elements []interface{}

	elements, err = m.DbProvider.DbGetAll("images")

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
