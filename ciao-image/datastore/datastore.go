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
	"sync"

	"github.com/01org/ciao/ciao-image/service"
)

// Datastore implements the datastore interface for the ciao-image service
//
// TBD: This is an in-memory store only, persistence needs to be added.
type Datastore struct {
	images     map[string]service.Image
	imagesLock *sync.RWMutex
}

func New() service.Datastore {
	var ds = &Datastore{}

	ds.Init()

	return ds
}

// Init initializes the datastore struct and must be called before anything.
func (d *Datastore) Init() {
	d.images = make(map[string]service.Image)
	d.imagesLock = &sync.RWMutex{}
}

// Create will add an image to the datastore.
func (d *Datastore) Create(i service.Image) error {
	d.imagesLock.Lock()
	d.images[i.ID] = i
	d.imagesLock.Unlock()
	return nil
}

// RetrieveAll gets returns all the known images.
func (d *Datastore) RetrieveAll() ([]service.Image, error) {
	var images []service.Image

	d.imagesLock.RLock()

	for _, i := range d.images {
		images = append(images, i)
	}

	d.imagesLock.RUnlock()

	return images, nil
}

// Retrieve returns the image specified by the ID string.
func (d *Datastore) Retrieve(ID string) (service.Image, error) {
	d.imagesLock.RLock()
	i, ok := d.images[ID]
	d.imagesLock.RUnlock()

	if !ok {
		return service.Image{}, service.ErrNoImage
	}

	return i, nil
}

// Update will modify an existing image.
func (d *Datastore) Update(i service.Image) error {
	d.imagesLock.Lock()

	_, ok := d.images[i.ID]
	if ok {
		d.images[i.ID] = i
	}

	d.imagesLock.Unlock()

	if !ok {
		return service.ErrNoImage
	}

	return nil
}

// Delete will delete an existing image.
func (d *Datastore) Delete(ID string) error {
	d.imagesLock.Lock()

	_, ok := d.images[ID]
	if ok {
		delete(d.images, ID)
	}

	d.imagesLock.Unlock()

	if !ok {
		return service.ErrNoImage
	}

	return nil
}
