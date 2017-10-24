//
// Copyright (c) 2017 Intel Corporation
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
//

package client

import (
	"fmt"
	"io"
	"net/http"

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/pkg/errors"
)

// GetImage retrieves the details for an image
func (client *Client) GetImage(imageID string) (types.Image, error) {
	var i types.Image

	var url string
	if client.IsPrivileged() && client.TenantID == "admin" {
		url = client.buildCiaoURL("images/%s", imageID)
	} else {
		url = client.buildCiaoURL("%s/images/%s", client.TenantID, imageID)
	}

	err := client.getResource(url, api.ImagesV1, nil, &i)

	return i, err
}

func (client *Client) uploadTenantImage(tenant, image string, data io.Reader) error {
	var url string
	if client.IsPrivileged() && client.TenantID == "admin" {
		url = client.buildCiaoURL("images/%s/file", image)
	} else {
		url = client.buildCiaoURL("%s/images/%s/file", client.TenantID, image)
	}

	resp, err := client.sendHTTPRequest("PUT", url, nil, data, fmt.Sprintf("%s/octet-stream", api.ImagesV1))
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Unexpected HTTP response code (%d): %s", resp.StatusCode, resp.Status)
	}

	return err
}

// CreateImage creates and uploads a new image
func (client *Client) CreateImage(name string, visibility types.Visibility, ID string, data io.Reader) (string, error) {
	opts := api.CreateImageRequest{
		Name:       name,
		ID:         ID,
		Visibility: visibility,
	}

	var url string
	if client.IsPrivileged() && client.TenantID == "admin" {
		url = client.buildCiaoURL("images")
	} else {
		url = client.buildCiaoURL("%s/images", client.TenantID)
	}

	var image types.Image
	err := client.postResource(url, api.ImagesV1, &opts, &image)
	if err != nil {
		return "", errors.Wrap(err, "Error creating image resource")
	}

	err = client.uploadTenantImage(client.TenantID, image.ID, data)
	if err != nil {
		return "", errors.Wrap(err, "Error uploading image data")
	}

	return image.ID, nil
}

// ListImages retrieves the set of available images
func (client *Client) ListImages() ([]types.Image, error) {
	var images []types.Image

	var url string
	if client.IsPrivileged() && client.TenantID == "admin" {
		url = client.buildCiaoURL("images")
	} else {
		url = client.buildCiaoURL("%s/images", client.TenantID)
	}

	err := client.getResource(url, api.ImagesV1, nil, &images)

	return images, err
}

// DeleteImage deletes the given image
func (client *Client) DeleteImage(imageID string) error {
	var url string
	if client.IsPrivileged() && client.TenantID == "admin" {
		url = client.buildCiaoURL("images/%s", imageID)
	} else {
		url = client.buildCiaoURL("%s/images/%s", client.TenantID, imageID)
	}

	return client.deleteResource(url, api.ImagesV1)
}
