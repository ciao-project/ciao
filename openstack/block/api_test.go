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
	"testing"

	"github.com/ciao-project/ciao/ciao-controller/types"
	storage "github.com/ciao-project/ciao/ciao-storage"
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
		`{"size": 10,"source_volid": null,"description":null,"name":null,"imageRef":null}`,
		http.StatusAccepted,
		`{"id":"new-test-id","bootable":false,"boot_index":0,"ephemeral":false,"local":false,"swap":false,"size":123456,"tenant_id":"test-tenant-id","state":"available","created":"0001-01-01T00:00:00Z","name":"new volume","description":"newly created volume","internal":false}`,
	},
	{
		"GET",
		"/v2/validtenantid/volumes",
		listVolumesDetail,
		"",
		http.StatusOK,
		`[{"id":"new-test-id","bootable":false,"boot_index":0,"ephemeral":false,"local":false,"swap":false,"size":123456,"tenant_id":"test-tenant-id","state":"available","created":"0001-01-01T00:00:00Z","name":"my volume","description":"my volume for stuff","internal":false},{"id":"new-test-id2","bootable":false,"boot_index":0,"ephemeral":false,"local":false,"swap":false,"size":123456,"tenant_id":"test-tenant-id","state":"available","created":"0001-01-01T00:00:00Z","name":"volume 2","description":"my other volume","internal":false}]`,
	},
	{
		"GET",
		"/v2/validtenantid/volumes/validvolumeid",
		showVolumeDetails,
		"",
		http.StatusOK,
		`{"id":"new-test-id","bootable":false,"boot_index":0,"ephemeral":false,"local":false,"swap":false,"size":123456,"tenant_id":"test-tenant-id","state":"available","created":"0001-01-01T00:00:00Z","name":"my volume","description":"my volume for stuff","internal":false}`,
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

func (vs testVolumeService) ShowVolumeDetails(tenant string, volume string) (types.BlockData, error) {
	return types.BlockData{
		BlockDevice: storage.BlockDevice{
			ID:   "new-test-id",
			Size: 123456,
		},
		State:       types.Available,
		Name:        "my volume",
		Description: "my volume for stuff",
		TenantID:    "test-tenant-id",
	}, nil
}

func (vs testVolumeService) CreateVolume(tenant string, req RequestedVolume) (types.BlockData, error) {
	return types.BlockData{
		BlockDevice: storage.BlockDevice{
			ID:   "new-test-id",
			Size: 123456,
		},
		State:       types.Available,
		Name:        "new volume",
		Description: "newly created volume",
		TenantID:    "test-tenant-id",
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

func (vs testVolumeService) ListVolumesDetail(tenant string) ([]types.BlockData, error) {
	return []types.BlockData{
		{
			BlockDevice: storage.BlockDevice{
				ID:   "new-test-id",
				Size: 123456,
			},
			State:       types.Available,
			Name:        "my volume",
			Description: "my volume for stuff",
			TenantID:    "test-tenant-id",
		},
		{
			BlockDevice: storage.BlockDevice{
				ID:   "new-test-id2",
				Size: 123456,
			},
			State:       types.Available,
			Name:        "volume 2",
			Description: "my other volume",
			TenantID:    "test-tenant-id",
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
