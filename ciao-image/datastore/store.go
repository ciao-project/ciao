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

const (
	tableImageMap = "images"
)

// ImageMap stores Image metadata
type ImageMap struct {
	sync.RWMutex
	m map[string]Image
}

// NewTable creates a new map of Images
func (i *ImageMap) NewTable() {
	i.m = make(map[string]Image)
}

// Name provides Images table name
func (i *ImageMap) Name() string {
	return tableImageMap
}

// NewElement generates a new Image struct
func (i *ImageMap) NewElement() interface{} {
	return &Image{}
}

// ImageStore is an image metadata cache.
type ImageStore struct {
	metaDs MetaDataStore
	rawDs  RawDataStore
	ImageMap
}

// Init initializes the datastore struct and must be called before anything.
func (s *ImageStore) Init(rawDs RawDataStore, metaDs MetaDataStore) error {
	s.ImageMap.NewTable()
	s.metaDs = metaDs
	s.rawDs = rawDs

	return nil
}

// CreateImage will add an image to the datastore.
func (s *ImageStore) CreateImage(i Image) error {
	defer s.ImageMap.Unlock()
	s.ImageMap.Lock()

	err := s.metaDs.Write(i)
	if err != nil {
		return err
	}

	return nil
}

// GetAllImages gets returns all the known images.
func (s *ImageStore) GetAllImages() ([]Image, error) {
	var images []Image
	defer s.ImageMap.RUnlock()
	s.ImageMap.RLock()

	images, err := s.metaDs.GetAll()
	if err != nil {
		return nil, err
	}

	return images, nil
}

// GetImage returns the image specified by the ID string.
func (s *ImageStore) GetImage(ID string) (Image, error) {
	defer s.ImageMap.RUnlock()
	s.ImageMap.RLock()

	img, err := s.metaDs.Get(ID)
	if err != nil {
		return Image{}, image.ErrNoImage
	}

	return img, nil
}

// UpdateImage will modify an existing image.
func (s *ImageStore) UpdateImage(i Image) error {
	defer s.ImageMap.Unlock()
	s.ImageMap.Lock()

	err := s.metaDs.Write(i)
	if err != nil {
		return err
	}

	return nil
}

// DeleteImage will delete an existing image.
func (s *ImageStore) DeleteImage(ID string) error {
	defer s.ImageMap.Unlock()
	s.ImageMap.Lock()

	err := s.metaDs.Delete(ID)
	if err != nil {
		return image.ErrNoImage
	}

	err = s.rawDs.Delete(ID)
	if err != nil {
		return err
	}

	return nil
}

// UploadImage will read an image, save it and update the image cache.
func (s *ImageStore) UploadImage(ID string, body io.Reader) error {
	img, err := s.metaDs.Get(ID)
	if err != nil {
		return image.ErrNoImage
	}

	if img.State == Saving {
		return image.ErrImageSaving
	}

	img.State = Saving

	if s.rawDs != nil {
		_, err := s.rawDs.Write(ID, body)
		if err != nil {
			return err
		}
	}

	img.State = Active
	s.ImageMap.Lock()

	err = s.metaDs.Write(img)
	if err != nil {
		return err
	}
	s.ImageMap.Unlock()

	return nil
}
