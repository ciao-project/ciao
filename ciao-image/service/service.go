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

package service

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/01org/ciao/openstack/identity"
	"github.com/01org/ciao/openstack/image"
	"github.com/01org/ciao/ssntp/uuid"
	"github.com/gorilla/mux"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
)

// State represents the state of the image.
type State string

const (
	// Created means that an empty image has been created
	Created State = "created"
)

func stateToStatus(state State) image.Status {
	switch state {
	case Created:
		return image.Queued
	}

	return image.Active
}

func visibility(i Image) image.Visibility {
	if i.TenantID == "" {
		return image.Public
	}
	return image.Private
}

// Type represents the valid image types.
type Type string

const (
	// Raw is the raw image format.
	Raw Type = "raw"

	// QCow is the qcow2 format.
	QCow Type = "qcow2"

	// ISO is the iso format.
	ISO Type = "iso"
)

// Image contains the information that ciao will store about the image
type Image struct {
	ID         string
	State      State
	TenantID   string
	Name       string
	CreateTime time.Time
	Type       Type
}

var (
	// ErrNoImage is returned when an image is not found.
	ErrNoImage = errors.New("Image not found")
)

// ImageService is the context for the image service implementation.
type ImageService struct {
	cache imageCache
}

// CreateImage will create an empty image in the image datastore.
func (is ImageService) CreateImage(req image.CreateImageRequest) (image.CreateImageResponse, error) {
	// create an ImageInfo struct and store it in our image
	// datastore.
	i := Image{
		ID:         uuid.Generate().String(),
		State:      Created,
		Name:       req.Name,
		CreateTime: time.Now(),
	}

	err := is.cache.createImage(i)
	if err != nil {
		return image.CreateImageResponse{}, err
	}

	return image.CreateImageResponse{
		Status:     image.Queued,
		CreatedAt:  i.CreateTime,
		Tags:       make([]string, 0),
		Locations:  make([]string, 0),
		DiskFormat: image.Raw,
		Visibility: visibility(i),
		Self:       fmt.Sprintf("/v2/images/%s", i.ID),
		Protected:  false,
		ID:         i.ID,
		File:       fmt.Sprintf("/v2/images/%s/file", i.ID),
		Schema:     "/v2/schemas/image",
		Name:       &i.Name,
	}, nil
}

// ListImages will return a list of all the images in the datastore.
func (is ImageService) ListImages() ([]image.CreateImageResponse, error) {
	var response []image.CreateImageResponse

	images, err := is.cache.getAllImages()
	if err != nil {
		return response, err
	}

	for _, img := range images {
		i := image.CreateImageResponse{
			Status:     stateToStatus(img.State),
			CreatedAt:  img.CreateTime,
			Tags:       make([]string, 0),
			Locations:  make([]string, 0),
			DiskFormat: image.DiskFormat(img.Type),
			Visibility: visibility(img),
			Self:       fmt.Sprintf("/v2/images/%s", img.ID),
			Protected:  false,
			ID:         img.ID,
			File:       fmt.Sprintf("/v2/images/%s/file", img.ID),
			Schema:     "/v2/schemas/image",
			Name:       &img.Name,
		}

		response = append(response, i)
	}

	return response, nil
}

// Config is required to setup the API context for the image service.
type Config struct {
	// Port represents the http port that should be used for the service.
	Port int

	// HTTPSCACert is the path to the http ca cert to use.
	HTTPSCACert string

	// HTTPSKey is the path to the https cert key.
	HTTPSKey string

	// Datastore is an interface to a persistent datastore.
	Datastore Datastore

	// IdentityEndpoint is the location of the keystone service.
	IdentityEndpoint string

	// Username is the service username for the image service in keystone.
	Username string

	// Password is the password for the image service user in keystone.
	Password string
}

func getIdentityClient(config Config) (*gophercloud.ServiceClient, error) {
	opt := gophercloud.AuthOptions{
		IdentityEndpoint: config.IdentityEndpoint + "v3/",
		Username:         config.Username,
		Password:         config.Password,
		TenantName:       "service",
		DomainID:         "default",
		AllowReauth:      true,
	}
	provider, err := openstack.AuthenticatedClient(opt)
	if err != nil {
		return nil, err
	}

	v3client := openstack.NewIdentityV3(provider)
	if v3client == nil {
		return nil, errors.New("Unable to get keystone V3 client")
	}

	return v3client, nil
}

// Start will get the Image API endpoints from the OpenStack image api,
// then wrap them in keystone validation. It will then start the https
// service.
func Start(config Config) error {
	is := ImageService{}
	err := is.cache.init(config.Datastore)
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
	validServices := []identity.ValidService{
		{ServiceType: "image", ServiceName: "ciao"},
		{ServiceType: "image", ServiceName: "glance"},
	}

	validAdmins := []identity.ValidAdmin{
		{Project: "service", Role: "admin"},
		{Project: "admin", Role: "admin"},
	}

	client, err := getIdentityClient(config)
	if err != nil {
		return err
	}

	err = r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		h := identity.Handler{
			Client:        client,
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
