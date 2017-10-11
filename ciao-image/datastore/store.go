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
	"fmt"
	"io"
	"sync"

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/clogger/gloginterface"
	"github.com/ciao-project/ciao/database"
)

const (
	tableImageMap = "images"
)

// ImageMap provide Image empty struct generator and mutex control
type ImageMap struct {
	sync.RWMutex
	m map[string]*types.Image
}

//NewTable creates a new map
func (i *ImageMap) NewTable() {
	i.m = make(map[string]*types.Image)
}

//Name provides the name of the map
func (i *ImageMap) Name() string {
	return tableImageMap
}

// NewElement generates a new Image struct
func (i *ImageMap) NewElement() interface{} {
	return &types.Image{}
}

//Add adds a value to the map with the specified key
func (i *ImageMap) Add(k string, v interface{}) error {
	val, ok := v.(*types.Image)
	if !ok {
		return fmt.Errorf("Invalid value type %t", v)
	}
	i.m[k] = val
	return nil
}

// ImageStore is an image metadata cache.
type ImageStore struct {
	metaDs MetaDataStore
	rawDs  RawDataStore
	ImageMap
}

// Init initializes the datastore struct and must be called before anything.
func (s *ImageStore) Init(rawDs RawDataStore, metaDs MetaDataStore) error {
	s.metaDs = metaDs
	s.rawDs = rawDs

	database.Logger = gloginterface.CiaoGlogLogger{}

	return nil
}

// Shutdown is the opposite of Init()
func (s *ImageStore) Shutdown() error {
	return s.metaDs.Shutdown()
}

// CreateImage will add an image to the datastore.
func (s *ImageStore) CreateImage(i types.Image) error {
	s.ImageMap.Lock()
	defer s.ImageMap.Unlock()

	err := s.metaDs.Write(i)
	if err != nil {
		return err
	}

	return nil
}

// GetAllImages gets returns all the known images.
func (s *ImageStore) GetAllImages(tenant string) ([]types.Image, error) {
	var images []types.Image
	s.ImageMap.RLock()
	defer s.ImageMap.RUnlock()
	images, err := s.metaDs.GetAll(tenant)
	if err != nil {
		return nil, err
	}

	return images, nil
}

// GetImage returns the image specified by the ID string.
func (s *ImageStore) GetImage(tenant, ID string) (types.Image, error) {
	s.ImageMap.RLock()
	defer s.ImageMap.RUnlock()

	img, err := s.metaDs.Get(tenant, ID)
	if err != nil {
		return types.Image{}, api.ErrNoImage
	}

	return img, nil
}

// UpdateImage will modify an existing image.
func (s *ImageStore) UpdateImage(i types.Image) error {
	s.ImageMap.Lock()
	defer s.ImageMap.Unlock()

	err := s.metaDs.Write(i)
	if err != nil {
		return err
	}

	return nil
}

// DeleteImage will delete an existing image.
func (s *ImageStore) DeleteImage(tenant, ID string) error {
	s.ImageMap.Lock()
	defer s.ImageMap.Unlock()

	img, err := s.metaDs.Get(tenant, ID)
	if err != nil {
		return err
	}

	if img == (types.Image{}) || img.TenantID != tenant {
		return api.ErrNoImage
	}

	if img.State == types.Active {
		err = s.rawDs.Delete(ID)
		if err != nil {
			return err
		}
	}

	if img.Visibility == types.Public {
		tenant = string(types.Public)
	}
	err = s.metaDs.Delete(tenant, ID)

	return err
}

// UploadImage will read an image, save it and update the image cache.
func (s *ImageStore) UploadImage(tenant, ID string, body io.Reader) error {
	s.ImageMap.RLock()
	img, err := s.metaDs.Get(tenant, ID)
	s.ImageMap.RUnlock()
	if err != nil {
		return err
	}

	if img == (types.Image{}) {
		return api.ErrNoImage
	}

	if img.State == types.Saving {
		return api.ErrImageSaving
	}

	img.State = types.Saving

	if s.rawDs != nil {
		err = s.rawDs.Write(ID, body)
		if err != nil {
			img.State = types.Killed
		}

		img.Size, err = s.rawDs.GetImageSize(ID)
		if err != nil {
			img.State = types.Killed
		}
	}

	if err == nil {
		img.State = types.Active
	}

	s.ImageMap.Lock()
	defer s.ImageMap.Unlock()
	metaDsErr := s.metaDs.Write(img)

	if err == nil && metaDsErr != nil {
		err = metaDsErr
	}

	return err
}
