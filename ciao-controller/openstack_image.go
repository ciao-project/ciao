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
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	imageDatastore "github.com/01org/ciao/ciao-image/datastore"
	"github.com/01org/ciao/ciao-storage"
	"github.com/01org/ciao/database"
	osIdentity "github.com/01org/ciao/openstack/identity"
	"github.com/01org/ciao/openstack/image"
	"github.com/01org/ciao/ssntp/uuid"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

// ImageService is the context for the image service implementation.
type ImageService struct {
	ds imageDatastore.DataStore
}

// CreateImage will create an empty image in the image datastore.
func (is ImageService) CreateImage(req image.CreateImageRequest) (image.DefaultResponse, error) {
	// create an ImageInfo struct and store it in our image
	// datastore.
	id := req.ID
	if id == "" {
		id = uuid.Generate().String()
	} else {
		if _, err := uuid.Parse(id); err != nil {
			return image.DefaultResponse{}, image.ErrBadUUID
		}

		img, err := is.ds.GetImage(id)
		if err != nil {
			return image.DefaultResponse{}, image.ErrDecodeImage
		}

		if img != (imageDatastore.Image{}) {
			return image.DefaultResponse{}, image.ErrAlreadyExists
		}
	}

	i := imageDatastore.Image{
		ID:         id,
		State:      imageDatastore.Created,
		Name:       req.Name,
		CreateTime: time.Now(),
	}

	err := is.ds.CreateImage(i)
	if err != nil {
		return image.DefaultResponse{}, err
	}

	return image.DefaultResponse{
		Status:     image.Queued,
		CreatedAt:  i.CreateTime,
		Tags:       make([]string, 0),
		Locations:  make([]string, 0),
		DiskFormat: image.Raw,
		Visibility: i.Visibility(),
		Self:       fmt.Sprintf("/v2/images/%s", i.ID),
		Protected:  false,
		ID:         i.ID,
		File:       fmt.Sprintf("/v2/images/%s/file", i.ID),
		Schema:     "/v2/schemas/image",
		Name:       &i.Name,
	}, nil
}

func createImageResponse(img imageDatastore.Image) (image.DefaultResponse, error) {
	return image.DefaultResponse{
		Status:     img.State.Status(),
		CreatedAt:  img.CreateTime,
		Tags:       make([]string, 0),
		Locations:  make([]string, 0),
		DiskFormat: image.DiskFormat(img.Type),
		Visibility: img.Visibility(),
		Self:       fmt.Sprintf("/v2/images/%s", img.ID),
		Protected:  false,
		ID:         img.ID,
		File:       fmt.Sprintf("/v2/images/%s/file", img.ID),
		Schema:     "/v2/schemas/image",
		Name:       &img.Name,
	}, nil
}

// ListImages will return a list of all the images in the datastore.
func (is ImageService) ListImages() ([]image.DefaultResponse, error) {
	var response []image.DefaultResponse

	images, err := is.ds.GetAllImages()
	if err != nil {
		return response, err
	}

	for _, img := range images {
		i, _ := createImageResponse(img)
		response = append(response, i)
	}

	return response, nil
}

// UploadImage will upload a raw image data and update its status.
func (is ImageService) UploadImage(imageID string, body io.Reader) (image.NoContentImageResponse, error) {
	var response image.NoContentImageResponse

	err := is.ds.UploadImage(imageID, body)
	if err != nil {
		return response, err
	}

	response.ImageID = imageID
	return response, nil
}

// DeleteImage will delete a raw image and its metadata
func (is ImageService) DeleteImage(imageID string) (image.NoContentImageResponse, error) {
	var response image.NoContentImageResponse

	err := is.ds.DeleteImage(imageID)
	if err != nil {
		return response, err
	}

	response.ImageID = imageID
	return response, nil
}

// GetImage will get the raw image data
func (is ImageService) GetImage(imageID string) (image.DefaultResponse, error) {
	var response image.DefaultResponse

	img, err := is.ds.GetImage(imageID)
	if err != nil {
		return response, err
	}

	if (img == imageDatastore.Image{}) {
		return response, image.ErrNoImage
	}

	response, _ = createImageResponse(img)
	return response, nil
}

// ImageConfig is required to setup the API context for the image service.
type ImageConfig struct {
	// Port represents the http port that should be used for the service.
	Port int

	// HTTPSCACert is the path to the http ca cert to use.
	HTTPSCACert string

	// HTTPSKey is the path to the https cert key.
	HTTPSKey string

	// DataStore is an interface to a persistent datastore for the image raw data.
	RawDataStore imageDatastore.RawDataStore

	// MetaDataStore is an interface to a persistent datastore for the image meta data.
	MetaDataStore imageDatastore.MetaDataStore
}

// startImageService will get the Image API endpoints from the OpenStack image api,
// then wrap them in keystone validation. It will then start the https
// service.
func (c *controller) startImageService() error {

	dbDir := filepath.Dir(*imageDatastoreLocation)
	dbFile := filepath.Base(*imageDatastoreLocation)
	glog.Infof("Image Datastore Location: %v", *imageDatastoreLocation)

	metaDs := &imageDatastore.MetaDs{
		DbProvider: database.NewBoltDBProvider(),
		DbDir:      dbDir,
		DbFile:     dbFile,
	}
	metaDsTables := []string{"images"}

	err := metaDs.DbInit(metaDs.DbDir, metaDs.DbFile)
	if err != nil {
		glog.Fatalf("Error on DB Initialization:%v ", err)
	}
	defer metaDs.DbClose()

	err = metaDs.DbTablesInit(metaDsTables)
	if err != nil {
		glog.Fatalf("Error on DB Tables Initialization:%v ", err)
	}

	rawDs := &imageDatastore.Ceph{
		ImageTempDir: c.image.MountPoint,
		BlockDriver: storage.CephDriver{
			ID: *cephID,
		},
	}

	config := ImageConfig{
		Port:          image.APIPort,
		HTTPSCACert:   httpsCAcert,
		HTTPSKey:      httpsKey,
		RawDataStore:  rawDs,
		MetaDataStore: metaDs,
	}

	is := ImageService{ds: &imageDatastore.ImageStore{}}
	err = is.ds.Init(config.RawDataStore, config.MetaDataStore)
	if err != nil {
		return err
	}

	apiConfig := image.APIConfig{
		Port:         config.Port,
		ImageService: is,
	}

	// get our routes.
	r := image.Routes(apiConfig)

	// setup identity for these routes.
	validServices := []osIdentity.ValidService{
		{ServiceType: "image", ServiceName: "ciao"},
		{ServiceType: "image", ServiceName: "glance"},
	}

	validAdmins := []osIdentity.ValidAdmin{
		{Project: "service", Role: "admin"},
		{Project: "admin", Role: "admin"},
	}

	err = r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		h := osIdentity.Handler{
			Client:        c.id.scV3,
			Next:          route.GetHandler(),
			ValidServices: validServices,
			ValidAdmins:   validAdmins,
		}

		route.Handler(h)

		return nil
	})
	if err != nil {
		return err
	}

	// start service.
	service := fmt.Sprintf(":%d", config.Port)

	return http.ListenAndServeTLS(service, config.HTTPSCACert, config.HTTPSKey, r)
}
