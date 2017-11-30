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
	"io/ioutil"
	"os"
	"regexp"
	"time"

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/payloads"
	"github.com/ciao-project/ciao/uuid"
	"github.com/golang/glog"
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
	}

	r := regexp.MustCompile("^[a-z0-9-.]{1,64}$")
	if !r.MatchString(req.Name) {
		return types.Image{}, types.ErrBadName
	}

	i := types.Image{
		ID:         id,
		TenantID:   tenantID,
		State:      types.Created,
		Name:       req.Name,
		CreateTime: time.Now(),
		Visibility: req.Visibility,
	}

	err := c.ds.AddImage(i)
	if err != nil {
		glog.Errorf("Error adding image to datastore: %v", err)
		return types.Image{}, err
	}

	res := <-c.qs.Consume(tenantID, payloads.RequestedResource{Type: payloads.Image, Value: 1})
	if !res.Allowed() {
		_ = c.ds.DeleteImage(id)
		c.qs.Release(tenantID, payloads.RequestedResource{Type: payloads.Image, Value: 1})
		return types.Image{}, api.ErrQuota
	}

	glog.Infof("Image %v added", id)
	return i, nil
}

// ListImages will return a list of all the images in the datastore.
func (c *controller) ListImages(tenant string) ([]types.Image, error) {
	glog.Infof("Listing images from [%v]", tenant)

	if tenant == "admin" {
		return c.ds.GetImages("", true)
	}

	return c.ds.GetImages(tenant, false)
}

func (c *controller) uploadImage(imageID string, body io.Reader) error {
	f, err := ioutil.TempFile("", "ciao-image")
	if err != nil {
		return fmt.Errorf("Error creating temporary image file: %v", err)
	}
	defer func() { _ = os.Remove(f.Name()) }()

	buf := make([]byte, 1<<16)
	_, err = io.CopyBuffer(f, body, buf)
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("Error writing to temporary image file: %v", err)
	}

	err = f.Close()
	if err != nil {
		return fmt.Errorf("Error closing temporary image file: %v", err)
	}

	_, err = c.CreateBlockDevice(imageID, f.Name(), 0)
	if err != nil {
		return fmt.Errorf("Error creating block device: %v", err)
	}

	err = c.CreateBlockDeviceSnapshot(imageID, "ciao-image")
	if err != nil {
		_ = c.DeleteBlockDevice(imageID)
		return fmt.Errorf("Unable to create snapshot: %v", err)
	}

	return nil
}

// UploadImage will upload a raw image data and update its status.
func (c *controller) UploadImage(tenantID, imageID string, body io.Reader) error {
	glog.Infof("Uploading image: %v", imageID)

	image, err := c.ds.GetImage(imageID)
	if err != nil {
		return err
	}

	if tenantID != "admin" && image.TenantID != image.TenantID {
		return api.ErrNoImage
	}

	image.State = types.Saving
	err = c.ds.UpdateImage(image)
	if err != nil {
		return err
	}

	err = c.uploadImage(imageID, body)
	if err != nil {
		glog.Errorf("Error uploading image: %v", err)
		image.State = types.Killed
		_ = c.ds.UpdateImage(image)
		return api.ErrImageSaving
	}

	imageSize, err := c.GetBlockDeviceSize(imageID)
	if err != nil {
		glog.Errorf("Error getting block device size: %v", err)
		image.State = types.Killed
		_ = c.ds.UpdateImage(image)
		return api.ErrImageSaving
	}

	image.Size = imageSize
	image.State = types.Active

	err = c.ds.UpdateImage(image)
	if err != nil {
		return err
	}

	glog.Infof("Image %v uploaded", imageID)
	return nil
}

// DeleteImage will delete a raw image and its metadata
func (c *controller) DeleteImage(tenantID, imageID string) error {
	glog.Infof("Deleting image: %v", imageID)

	image, err := c.ds.GetImage(imageID)
	if err != nil {
		return err
	}

	if tenantID != "admin" && image.TenantID != image.TenantID {
		return api.ErrNoImage
	}

	err = c.ds.DeleteImage(imageID)
	if err != nil {
		return err
	}

	c.qs.Release(tenantID, payloads.RequestedResource{Type: payloads.Image, Value: 1})

	err = c.DeleteBlockDeviceSnapshot(imageID, "ciao-image")
	if err != nil {
		return fmt.Errorf("Unable to delete snapshot: %v", err)
	}

	err = c.DeleteBlockDevice(imageID)
	if err != nil {
		return fmt.Errorf("Error deleting block device: %v", err)
	}

	glog.Infof("Image %v deleted", imageID)
	return nil
}

// GetImage gets image metadata after checking permissions
func (c *controller) GetImage(tenantID, imageID string) (types.Image, error) {
	glog.Infof("Getting Image [%v] from [%v]", imageID, tenantID)

	id, err := c.ds.ResolveImage(tenantID, imageID)
	if err != nil {
		return types.Image{}, err
	}

	image, err := c.ds.GetImage(id)
	if err != nil {
		return types.Image{}, err
	}

	if tenantID != "admin" && image.TenantID != image.TenantID {
		return types.Image{}, api.ErrNoImage
	}

	glog.Infof("Image %v found", imageID)
	return image, nil
}
