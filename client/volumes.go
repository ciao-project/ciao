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
	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
)

// CreateVolume creates a volume from a request
func (client *Client) CreateVolume(req api.RequestedVolume) (types.Volume, error) {
	var vol types.Volume

	url := client.buildCiaoURL("%s/volumes", client.TenantID)
	err := client.postResource(url, api.VolumesV1, &req, &vol)

	return vol, err
}

// ListVolumes lists the volumes
func (client *Client) ListVolumes() ([]types.Volume, error) {
	var volumes []types.Volume

	url := client.buildCiaoURL("%s/volumes", client.TenantID)
	err := client.getResource(url, api.VolumesV1, nil, &volumes)

	return volumes, err
}

// GetVolume gets the details of a single volume
func (client *Client) GetVolume(volumeID string) (types.Volume, error) {
	var volume types.Volume

	url := client.buildCiaoURL("%s/volumes/%s", client.TenantID, volumeID)
	err := client.getResource(url, api.VolumesV1, nil, &volume)

	return volume, err
}

// DeleteVolume deletes a volume
func (client *Client) DeleteVolume(volumeID string) error {
	url := client.buildCiaoURL("%s/volumes/%s", client.TenantID, volumeID)
	return client.deleteResource(url, api.VolumesV1)
}

// AttachVolume attaches a volume to an instance
func (client *Client) AttachVolume(volumeID string, instanceID, mountPoint string, mode string) error {
	url := client.buildCiaoURL("%s/volumes/%s/action", client.TenantID, volumeID)

	type AttachRequest struct {
		MountPoint   string `json:"mountpoint"`
		Mode         string `json:"mode"`
		InstanceUUID string `json:"instance_uuid"`
	}

	// mountpoint or mode isn't required
	var attachReq = struct {
		Attach AttachRequest `json:"attach"`
	}{
		Attach: AttachRequest{
			MountPoint:   mountPoint,
			Mode:         mode,
			InstanceUUID: instanceID,
		},
	}

	err := client.postResource(url, api.VolumesV1, &attachReq, nil)

	return err
}

// DetachVolume detaches a volume from an instance
func (client *Client) DetachVolume(volumeID string) error {
	url := client.buildCiaoURL("%s/volumes/%s/action", client.TenantID, volumeID)

	type DetachRequest struct {
		AttachmentID string `json:"attachment_id,omitempty"`
	}
	var detachReq = struct {
		Detach DetachRequest `json:"detach"`
	}{
		Detach: DetachRequest{},
	}

	err := client.postResource(url, api.VolumesV1, &detachReq, nil)

	return err
}
