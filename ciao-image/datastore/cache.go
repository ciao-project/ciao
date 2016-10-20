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
	"io"
	"sync"

	"github.com/01org/ciao/openstack/image"
)

// ImageCache is an image metadata cache.
type ImageCache struct {
	images map[string]Image
	lock   *sync.RWMutex
	metaDs MetaDataStore
	rawDs  RawDataStore
}

// Init initializes the datastore struct and must be called before anything.
func (c *ImageCache) Init(rawDs RawDataStore, metaDs MetaDataStore) error {
	c.images = make(map[string]Image)
	c.lock = &sync.RWMutex{}
	c.metaDs = metaDs
	c.rawDs = rawDs

	return nil
}

// CreateImage will add an image to the datastore.
func (c *ImageCache) CreateImage(i Image) error {
	defer c.lock.Unlock()
	c.lock.Lock()

	c.images[i.ID] = i
	err := c.metaDs.Write(i)
	if err != nil {
		delete(c.images, i.ID)
		return err
	}

	return nil
}

// GetAllImages gets returns all the known images.
func (c *ImageCache) GetAllImages() ([]Image, error) {
	var images []Image

	defer c.lock.RUnlock()
	c.lock.RLock()

	metaDsImages, err := c.metaDs.GetAll()
	if err != nil {
		return nil, err
	}
	for _, img := range metaDsImages {
		c.images[img.ID] = img
	}

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
		img, err := c.metaDs.Get(ID)
		if err != nil {
			return Image{}, image.ErrNoImage
		}
		i = img
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
		return image.ErrNoImage
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
		err := c.rawDs.Delete(ID)
		if err != nil {
			return err
		}
	}

	err := c.metaDs.Delete(ID)
	if err != nil {
		return image.ErrNoImage
	}

	return nil
}

// UploadImage will read an image, save it and update the image cache.
func (c *ImageCache) UploadImage(ID string, body io.Reader) error {
	c.lock.Lock()

	img, ok := c.images[ID]
	if !ok {
		c.lock.Unlock()
		return image.ErrNoImage
	}

	if img.State == Saving {
		c.lock.Unlock()
		return image.ErrImageSaving
	}

	img.State = Saving

	c.lock.Unlock()

	if c.rawDs != nil {
		_, err := c.rawDs.Write(ID, body)
		if err != nil {
			return err
		}
	}

	c.lock.Lock()

	img.State = Active

	c.lock.Unlock()

	return nil
}
