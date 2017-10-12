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

package main

import (
	"io"
	"path/filepath"
	"time"

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
	imageDatastore "github.com/ciao-project/ciao/ciao-image/datastore"
	"github.com/ciao-project/ciao/ciao-storage"
	"github.com/ciao-project/ciao/database"
	"github.com/ciao-project/ciao/payloads"
	"github.com/ciao-project/ciao/uuid"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

// CreateImage will create an empty image in the image datastore.
func (c *controller) CreateImage(tenantID string, req api.CreateImageRequest) (types.Image, error) {
	// create an ImageInfo struct and store it in our image
	// datastore.
	glog.Infof("Creating Image: %v", req.ID)

	id := req.ID
	if id == "" {
		id = uuid.Generate().String()
	} else {
		if _, err := uuid.Parse(id); err != nil {
			glog.Errorf("Error on parsing UUID: %v", err)
			return types.Image{}, api.ErrBadUUID
		}

		img, _ := c.ids.GetImage(tenantID, id)
		if img != (types.Image{}) {
			glog.Errorf("Image [%v] already exists", id)
			return types.Image{}, api.ErrAlreadyExists
		}
	}

	i := types.Image{
		ID:         id,
		TenantID:   tenantID,
		State:      types.Created,
		Name:       req.Name,
		CreateTime: time.Now(),
		Visibility: req.Visibility,
	}

	err := c.ids.CreateImage(i)
	if err != nil {
		glog.Errorf("Error on creating image: %v", err)
		return types.Image{}, err
	}

	res := <-c.qs.Consume(tenantID, payloads.RequestedResource{Type: payloads.Image, Value: 1})
	if !res.Allowed() {
		c.ids.DeleteImage(tenantID, id)
		c.qs.Release(tenantID, payloads.RequestedResource{Type: payloads.Image, Value: 1})
		return types.Image{}, api.ErrQuota
	}

	glog.Infof("Image %v created", id)
	return i, nil
}

// ListImages will return a list of all the images in the datastore.
func (c *controller) ListImages(tenant string) ([]types.Image, error) {
	glog.Infof("Listing images from [%v]", tenant)
	response := []types.Image{}

	images, err := c.ids.GetAllImages(tenant)
	if err != nil {
		glog.Errorf("Error on retrieving images from tenant [%v]: %v", tenant, err)
		return response, err
	}

	return images, nil
}

// UploadImage will upload a raw image data and update its status.
func (c *controller) UploadImage(tenantID, imageID string, body io.Reader) error {
	glog.Infof("Uploading image: %v", imageID)

	err := c.ids.UploadImage(tenantID, imageID, body)
	if err != nil {
		glog.Errorf("Error on uploading image: %v", err)
		return err
	}

	glog.Infof("Image %v uploaded", imageID)
	return nil
}

// DeleteImage will delete a raw image and its metadata
func (c *controller) DeleteImage(tenantID, imageID string) error {
	glog.Infof("Deleting image: %v", imageID)

	err := c.ids.DeleteImage(tenantID, imageID)
	if err != nil {
		glog.Errorf("Error on deleting image: %v", err)
		return err
	}

	c.qs.Release(tenantID, payloads.RequestedResource{Type: payloads.Image, Value: 1})

	glog.Infof("Image %v deleted", imageID)
	return nil
}

// GetImage will get the raw image data
func (c *controller) GetImage(tenantID, imageID string) (types.Image, error) {
	glog.Infof("Getting Image [%v] from [%v]", imageID, tenantID)
	var response types.Image

	img, err := c.ids.GetImage(tenantID, imageID)
	if err != nil {
		glog.Errorf("Error on getting image: %v", err)
		return response, err
	}

	if (img == types.Image{}) {
		glog.Infof("Image %v not found", imageID)
		return response, api.ErrNoImage
	}

	glog.Infof("Image %v found", imageID)
	return img, nil
}

// Init initialises the image service
func (c *controller) InitImageDatastore() error {
	dbDir := filepath.Dir(*imageDatastoreLocation)
	dbFile := filepath.Base(*imageDatastoreLocation)

	metaDs := &imageDatastore.MetaDs{
		DbProvider: database.NewBoltDBProvider(),
		DbDir:      dbDir,
		DbFile:     dbFile,
	}

	glog.Info("ciao-image - MetaDatastore Initialization")
	glog.Infof("DBProvider : %T", metaDs.DbProvider)
	glog.Infof("DbDir      : %v", metaDs.DbDir)
	glog.Infof("DbFile     : %v", metaDs.DbFile)

	metaDsTables := []string{"public", "internal"}

	err := metaDs.DbInit(metaDs.DbDir, metaDs.DbFile)

	if err != nil {
		return errors.Wrap(err, "Error on DB Initialization")
	}

	err = metaDs.DbTablesInit(metaDsTables)
	if err != nil {
		return errors.Wrap(err, "Error on DB Tables Initialization")
	}

	rawDs := &imageDatastore.Ceph{
		ImageTempDir: *imagesPath,
		BlockDriver: storage.CephDriver{
			ID: *cephID,
		},
	}

	glog.Info("ciao-image - Initialize raw datastore")
	glog.Infof("rawDs        : %T", rawDs)
	glog.Infof("ImageTempDir : %v", rawDs.ImageTempDir)
	glog.Infof("ID           : %v", rawDs.BlockDriver.ID)

	config := ImageConfig{
		HTTPSCACert:   httpsCAcert,
		HTTPSKey:      httpsKey,
		RawDataStore:  rawDs,
		MetaDataStore: metaDs,
	}

	glog.Info("ciao-image - Configuration")
	glog.Infof("HTTPSCACert   : %v", config.HTTPSCACert)
	glog.Infof("HTTPSKey      : %v", config.HTTPSKey)
	glog.Infof("RawDataStore  : %T", config.RawDataStore)
	glog.Infof("MetaDataStore : %T", config.MetaDataStore)

	c.ids = &imageDatastore.ImageStore{}
	err = c.ids.Init(config.RawDataStore, config.MetaDataStore)
	if err != nil {
		return err
	}

	return nil
}

// ImageConfig is required to setup the API context for the image service.
type ImageConfig struct {
	// HTTPSCACert is the path to the http ca cert to use.
	HTTPSCACert string

	// HTTPSKey is the path to the https cert key.
	HTTPSKey string

	// DataStore is an interface to a persistent datastore for the image raw data.
	RawDataStore imageDatastore.RawDataStore

	// MetaDataStore is an interface to a persistent datastore for the image meta data.
	MetaDataStore imageDatastore.MetaDataStore
}
