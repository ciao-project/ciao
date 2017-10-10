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

package block

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

type test struct {
	method           string
	pattern          string
	handler          func(*Context, http.ResponseWriter, *http.Request) (APIResponse, error)
	request          string
	expectedStatus   int
	expectedResponse string
}

var tests = []test{
	{
		"POST",
		"/v2/validtenantid/volumes",
		createVolume,
		`{"volume":{"size": 10,"availability_zone": null,"source_volid": null,"description":null,"multiattach ":false,"snapshot_id":null,"name":null,"imageRef":null,"volume_type":null,"metadata":{},"source_replica":null,"consistencygroup_id":null}}`,
		http.StatusAccepted,
		`{"volume":{"status":"creating","user_id":"validuserid","attachments":[],"links":[],"bootable":"false","encrypted":false,"created_at":null,"updated_at":null,"replication_status":"disabled","multiattach":false,"metadata":{},"id":"validvolumeid","size":10}}`,
	},
	{
		"GET",
		"/v2/validtenantid/volumes",
		listVolumes,
		"",
		http.StatusOK,
		`{"volumes":[{"id":"validvolumeid1","links":[],"name":"vol-001"},{"id":"validvolumeid2","links":[],"name":"vol-002"},{"id":"validvolumeid3","links":[],"name":"vol-003"}]}`,
	},
	{
		"GET",
		"/v2/validtenantid/volumes/detail",
		listVolumesDetail,
		"",
		http.StatusOK,
		`{"volumes":[{"attachments":[{"server_id":"f4fda93b-06e0-4743-8117-bc8bcecd651b","attachment_id":"3b4db356-253d-4fab-bfa0-e3626c0b8405","host_name":"","volume_id":"6edbc2f4-1507-44f8-ac0d-eed1d2608d38","device":"/dev/vdb","id":"6edbc2f4-1507-44f8-ac0d-eed1d2608d38"}],"links":[{"href":"http://23.253.248.171:8776/v2/bab7d5c60cd041a0a36f7c4b6e1dd978/volumes/6edbc2f4-1507-44f8-ac0d-eed1d2608d38","rel":"self"},{"href":"http://23.253.248.171:8776/bab7d5c60cd041a0a36f7c4b6e1dd978/volumes/6edbc2f4-1507-44f8-ac0d-eed1d2608d38","rel":"bookmark"}],"availability_zone":"nova","os-vol-host-attr:host":"cephcluster","encrypted":false,"replication_status":"disabled","id":"6edbc2f4-1507-44f8-ac0d-eed1d2608d38","size":2,"user_id":"32779452fcd34ae1a53a797ac8a1e064","os-vol-tenant-attr:tenant_id":"bab7d5c60cd041a0a36f7c4b6e1dd978","metadata":{"attached_mode":"rw","readonly":false},"status":"in-use","multiattach":true,"name":"vol-001","bootable":"false","created_at":null}]}`,
	},
	{
		"GET",
		"/v2/validtenantid/volumes/validvolumeid",
		showVolumeDetails,
		"",
		http.StatusOK,
		`{"volume":{"links":[{"href":"http://localhost:8776/v2/0c2eba2c5af04d3f9e9d0d410b371fde/volumes/5aa119a8-d25b-45a7-8d1b-88e127885635","rel":"self"},{"href":"http://localhost:8776/0c2eba2c5af04d3f9e9d0d410b371fde/volumes/5aa119a8-d25b-45a7-8d1b-88e127885635","rel":"bookmark"}],"availability_zone":"nova","os-vol-host-attr:host":"ip-10-168-107-25","encrypted":false,"replication_status":"disabled","id":"5aa119a8-d25b-45a7-8d1b-88e127885635","size":1,"user_id":"32779452fcd34ae1a53a797ac8a1e064","os-vol-tenant-attr:tenant_id":"0c2eba2c5af04d3f9e9d0d410b371fde","metadata":{},"status":"available","description":"Super volume.","multiattach":false,"name":"vol-002","bootable":"false","created_at":null,"volume_type":"None"}}`,
	},
	{
		"DELETE",
		"/v2/validtenantid/volumes/validvolumeid",
		deleteVolume,
		"",
		http.StatusAccepted,
		"null",
	},
	{
		"POST",
		"/v2/validtenantid/volumes/validvolumeid/action",
		volumeAction,
		`{"os-attach":{"instance_uuid":"validinstanceid","mountpoint":"/dev/vdc"}}`,
		http.StatusAccepted,
		"null",
	},
	{
		"POST",
		"/v2/validtenantid/volumes/validvolumeid/action",
		volumeAction,
		`{"os-detach":{}}`,
		http.StatusAccepted,
		"null",
	},
}

type testVolumeService struct{}

func (vs testVolumeService) ShowVolumeDetails(tenant string, volume string) (VolumeDetail, error) {
	volName := "vol-002"

	selfLink := Link{
		Href: "http://localhost:8776/v2/0c2eba2c5af04d3f9e9d0d410b371fde/volumes/5aa119a8-d25b-45a7-8d1b-88e127885635",
		Rel:  "self",
	}

	bookLink := Link{
		Href: "http://localhost:8776/0c2eba2c5af04d3f9e9d0d410b371fde/volumes/5aa119a8-d25b-45a7-8d1b-88e127885635",
		Rel:  "bookmark",
	}

	zone := "nova"
	description := "Super volume."
	volType := "None"

	meta := map[string]interface{}{
		"contents": "not junk",
	}

	return VolumeDetail{
		Attachments:       make([]Attachment, 0),
		Links:             []Link{selfLink, bookLink},
		ReplicationStatus: ReplicationDisabled,
		ID:                "5aa119a8-d25b-45a7-8d1b-88e127885635",
		Status:            Available,
		Name:              volName,
		AvailabilityZone:  zone,
		OSVolHostAttr:     "ip-10-168-107-25",
		Size:              1,
		UserID:            "32779452fcd34ae1a53a797ac8a1e064",
		OSVolTenantAttr:   "0c2eba2c5af04d3f9e9d0d410b371fde",
		MetaData:          meta,
		MultiAttach:       false,
		VolumeType:        volType,
		Description:       description,
		Bootable:          strconv.FormatBool(false),
	}, nil
}

func (vs testVolumeService) CreateVolume(tenant string, req RequestedVolume) (Volume, error) {
	return Volume{
		Status:            Creating,
		UserID:            "validuserid",
		Attachments:       make([]Attachment, 0),
		Links:             make([]Link, 0),
		Bootable:          strconv.FormatBool(req.ImageRef != ""),
		Description:       req.Description,
		VolumeType:        req.VolumeType,
		Name:              req.Name,
		ReplicationStatus: ReplicationDisabled,
		SourceVolID:       req.SourceVolID,
		SnapshotID:        req.SnapshotID,
		MultiAttach:       req.MultiAttach,
		MetaData:          req.MetaData,
		ID:                "validvolumeid",
		Size:              req.Size,
	}, nil
}

func (vs testVolumeService) DeleteVolume(tenant string, volume string) error {
	return nil
}

func (vs testVolumeService) AttachVolume(tenant string, volume string, instance string, mountpoint string) error {
	return nil
}

func (vs testVolumeService) DetachVolume(tenant string, volume string, attachment string) error {
	return nil
}

func (vs testVolumeService) ListVolumes(tenant string) ([]ListVolume, error) {
	return []ListVolume{
		{"validvolumeid1", make([]Link, 0), "vol-001"},
		{"validvolumeid2", make([]Link, 0), "vol-002"},
		{"validvolumeid3", make([]Link, 0), "vol-003"},
	}, nil
}

func (vs testVolumeService) ListVolumesDetail(tenant string) ([]VolumeDetail, error) {
	volName := "vol-001"

	attachment := Attachment{
		ServerUUID:     "f4fda93b-06e0-4743-8117-bc8bcecd651b",
		AttachmentUUID: "3b4db356-253d-4fab-bfa0-e3626c0b8405",
		VolumeUUID:     "6edbc2f4-1507-44f8-ac0d-eed1d2608d38",
		Device:         "/dev/vdb",
		DeviceUUID:     "6edbc2f4-1507-44f8-ac0d-eed1d2608d38",
	}

	selfLink := Link{
		Href: "http://23.253.248.171:8776/v2/bab7d5c60cd041a0a36f7c4b6e1dd978/volumes/6edbc2f4-1507-44f8-ac0d-eed1d2608d38",
		Rel:  "self",
	}

	bookLink := Link{
		Href: "http://23.253.248.171:8776/bab7d5c60cd041a0a36f7c4b6e1dd978/volumes/6edbc2f4-1507-44f8-ac0d-eed1d2608d38",
		Rel:  "bookmark",
	}

	zone := "nova"

	meta := map[string]interface{}{
		"attached_mode": "rw",
		"readonly":      false,
	}

	return []VolumeDetail{
		{
			Attachments:       []Attachment{attachment},
			Links:             []Link{selfLink, bookLink},
			ReplicationStatus: ReplicationDisabled,
			ID:                "6edbc2f4-1507-44f8-ac0d-eed1d2608d38",
			Status:            InUse,
			Name:              volName,
			AvailabilityZone:  zone,
			OSVolHostAttr:     "cephcluster",
			Size:              2,
			UserID:            "32779452fcd34ae1a53a797ac8a1e064",
			OSVolTenantAttr:   "bab7d5c60cd041a0a36f7c4b6e1dd978",
			MetaData:          meta,
			MultiAttach:       true,
			Bootable:          strconv.FormatBool(false),
		},
	}, nil
}

func TestAPIResponse(t *testing.T) {
	var vs testVolumeService

	// TBD: add context to test definition so it can be created per
	// endpoint with either a pass testVolumeService or a failure
	// one.
	context := &Context{vs}

	for _, tt := range tests {
		req, err := http.NewRequest(tt.method, tt.pattern, bytes.NewBuffer([]byte(tt.request)))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := APIHandler{context, tt.handler}

		handler.ServeHTTP(rr, req)

		status := rr.Code
		if status != tt.expectedStatus {
			t.Errorf("got %v, expected %v", status, tt.expectedStatus)
		}

		if rr.Body.String() != tt.expectedResponse {
			t.Errorf("%s: failed\ngot: %v\nexp: %v", tt.pattern, rr.Body.String(), tt.expectedResponse)
		}
	}
}

func TestRoutes(t *testing.T) {
	var vs testVolumeService
	config := APIConfig{vs}

	r := Routes(config, nil)
	if r == nil {
		t.Fatalf("No routes returned")
	}
}
