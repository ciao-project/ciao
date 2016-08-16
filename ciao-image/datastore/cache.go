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
)

// ImageCache is an image metadata cache.
type ImageCache struct {
	images map[string]Image
	lock   *sync.RWMutex
	metaDs MetaDataStore
	ds     DataStore
}

// Init initializes the datastore struct and must be called before anything.
func (c *ImageCache) Init(ds DataStore, metaDs MetaDataStore) error {
	c.images = make(map[string]Image)
	c.lock = &sync.RWMutex{}
	c.metaDs = metaDs
	c.ds = ds

	return nil
}

// CreateImage will add an image to the datastore.
func (c *ImageCache) CreateImage(i Image) error {
	defer c.lock.Unlock()
	c.lock.Lock()

	c.images[i.ID] = i

	return nil
}

// GetAllImages gets returns all the known images.
func (c *ImageCache) GetAllImages() ([]Image, error) {
	var images []Image

	defer c.lock.RUnlock()
	c.lock.RLock()

	for _, i := range c.images {
		images = append(images, i)
	}

	return images, nil
}

// GetImage returns the image specified by the ID string.
func (c *ImageCache) GetImage(ID string) (Image, error) {
	defer c.lock.RUnlock()
	c.lock.RLock()

	i, ok := c.images[ID]

	if !ok {
		return Image{}, ErrNoImage
	}

	return i, nil
}

// UpdateImage will modify an existing image.
func (c *ImageCache) UpdateImage(i Image) error {
	defer c.lock.Unlock()
	c.lock.Lock()

	_, ok := c.images[i.ID]
	if ok {
		c.images[i.ID] = i
	}

	if !ok {
		return ErrNoImage
	}

	return nil
}

// DeleteImage will delete an existing image.
func (c *ImageCache) DeleteImage(ID string) error {
	defer c.lock.Unlock()
	c.lock.Lock()

	_, ok := c.images[ID]
	if ok {
		delete(c.images, ID)
	}

	if !ok {
		return ErrNoImage
	}

	return nil
}