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

package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/payloads"
	"github.com/ciao-project/ciao/service"
)

type test struct {
	method           string
	request          string
	requestBody      string
	media            string
	expectedStatus   int
	expectedResponse string
}

var tests = []test{
	{
		"GET",
		"/",
		"",
		"application/text",
		http.StatusOK,
		`[{"rel":"pools","href":"/pools","version":"x.ciao.pools.v1","minimum_version":"x.ciao.pools.v1"},{"rel":"external-ips","href":"/external-ips","version":"x.ciao.external-ips.v1","minimum_version":"x.ciao.external-ips.v1"},{"rel":"workloads","href":"/workloads","version":"x.ciao.workloads.v1","minimum_version":"x.ciao.workloads.v1"},{"rel":"tenants","href":"/tenants","version":"x.ciao.tenants.v1","minimum_version":"x.ciao.tenants.v1"},{"rel":"node","href":"/node","version":"x.ciao.node.v1","minimum_version":"x.ciao.node.v1"}]`,
	},
	{
		"GET",
		"/pools",
		"",
		fmt.Sprintf("application/%s", PoolsV1),
		http.StatusOK,
		`{"pools":[{"id":"ba58f471-0735-4773-9550-188e2d012941","name":"testpool","free":0,"total_ips":0,"links":[{"rel":"self","href":"/pools/ba58f471-0735-4773-9550-188e2d012941"}]}]}`,
	},
	{
		"GET",
		"/pools?name=testpool",
		"",
		fmt.Sprintf("application/%s", PoolsV1),
		http.StatusOK,
		`{"pools":[{"id":"ba58f471-0735-4773-9550-188e2d012941","name":"testpool","free":0,"total_ips":0,"links":[{"rel":"self","href":"/pools/ba58f471-0735-4773-9550-188e2d012941"}]}]}`,
	},
	{
		"POST",
		"/pools",
		`{"name":"testpool"}`,
		fmt.Sprintf("application/%s", PoolsV1),
		http.StatusNoContent,
		"null",
	},
	{
		"GET",
		"/pools/ba58f471-0735-4773-9550-188e2d012941",
		"",
		fmt.Sprintf("application/%s", PoolsV1),
		http.StatusOK,
		`{"id":"ba58f471-0735-4773-9550-188e2d012941","name":"testpool","free":0,"total_ips":0,"links":[{"rel":"self","href":"/pools/ba58f471-0735-4773-9550-188e2d012941"}],"subnets":[],"ips":[]}`,
	},
	{
		"DELETE",
		"/pools/ba58f471-0735-4773-9550-188e2d012941",
		"",
		fmt.Sprintf("application/%s", PoolsV1),
		http.StatusNoContent,
		"null",
	},
	{
		"POST",
		"/pools/ba58f471-0735-4773-9550-188e2d012941",
		`{"subnet":"192.168.0.0/24"}`,
		fmt.Sprintf("application/%s", PoolsV1),
		http.StatusNoContent,
		"null",
	},
	{
		"DELETE",
		"/pools/ba58f471-0735-4773-9550-188e2d012941/subnets/ba58f471-0735-4773-9550-188e2d012941",
		"",
		fmt.Sprintf("application/%s", PoolsV1),
		http.StatusNoContent,
		"null",
	},
	{
		"DELETE",
		"/pools/ba58f471-0735-4773-9550-188e2d012941/external-ips/ba58f471-0735-4773-9550-188e2d012941",
		"",
		fmt.Sprintf("application/%s", PoolsV1),
		http.StatusNoContent,
		"null",
	},
	{
		"GET",
		"/external-ips",
		"",
		fmt.Sprintf("application/%s", ExternalIPsV1),
		http.StatusOK,
		`[{"mapping_id":"ba58f471-0735-4773-9550-188e2d012941","external_ip":"192.168.0.1","internal_ip":"172.16.0.1","instance_id":"","tenant_id":"8a497c68-a88a-4c1c-be56-12a4883208d3","pool_id":"f384ffd8-e7bd-40c2-8552-2efbe7e3ad6e","pool_name":"mypool","links":[{"rel":"self","href":"/external-ips/ba58f471-0735-4773-9550-188e2d012941"},{"rel":"pool","href":"/pools/f384ffd8-e7bd-40c2-8552-2efbe7e3ad6e"}]}]`,
	},
	{
		"POST",
		"/19df9b86-eda3-489d-b75f-d38710e210cb/external-ips",
		`{"pool_name":"apool","instance_id":"validinstanceID"}`,
		fmt.Sprintf("application/%s", ExternalIPsV1),
		http.StatusNoContent,
		"null",
	},
	{
		"POST",
		"/workloads",
		`{"id":"","description":"testWorkload","fw_type":"legacy","vm_type":"qemu","image_name":"","config":"this will totally work!","defaults":[]}`,
		fmt.Sprintf("application/%s", WorkloadsV1),
		http.StatusCreated,
		`{"workload":{"id":"ba58f471-0735-4773-9550-188e2d012941","description":"testWorkload","fw_type":"legacy","vm_type":"qemu","image_name":"","config":"this will totally work!","defaults":[],"storage":null},"link":{"rel":"self","href":"/workloads/ba58f471-0735-4773-9550-188e2d012941"}}`,
	},
	{
		"DELETE",
		"/workloads/76f4fa99-e533-4cbd-ab36-f6c0f51292ed",
		"",
		fmt.Sprintf("application/%s", WorkloadsV1),
		http.StatusNoContent,
		"null",
	},
	{
		"GET",
		"/workloads/ba58f471-0735-4773-9550-188e2d012941",
		"",
		fmt.Sprintf("application/%s", WorkloadsV1),
		http.StatusOK,
		`{"id":"ba58f471-0735-4773-9550-188e2d012941","description":"testWorkload","fw_type":"legacy","vm_type":"qemu","image_name":"","config":"this will totally work!","defaults":null,"storage":null}`,
	},
	{
		"GET",
		"/workloads",
		"",
		fmt.Sprintf("application/%s", WorkloadsV1),
		http.StatusOK,
		`[{"id":"ba58f471-0735-4773-9550-188e2d012941","description":"testWorkload","fw_type":"legacy","vm_type":"qemu","image_name":"","config":"this will totally work!","defaults":null,"storage":null}]`,
	},
	{
		"GET",
		"/tenants/093ae09b-f653-464e-9ae6-5ae28bd03a22/quotas",
		"",
		fmt.Sprintf("application/%s", TenantsV1),
		http.StatusOK,
		`{"quotas":[{"name":"test-quota-1","value":"10","usage":"3"},{"name":"test-quota-2","value":"unlimited","usage":"10"},{"name":"test-limit","value":"123"}]}`,
	},
	{
		"GET",
		"/tenants",
		"",
		fmt.Sprintf("application/%s", TenantsV1),
		http.StatusOK,
		`{"tenants":[{"id":"bc70dcd6-7298-4933-98a9-cded2d232d02","name":"Test Tenant","links":[{"rel":"self","href":"/tenants/bc70dcd6-7298-4933-98a9-cded2d232d02"}]}]}`,
	},
	{
		"GET",
		"/tenants/093ae09b-f653-464e-9ae6-5ae28bd03a22",
		"",
		fmt.Sprintf("application/%s", TenantsV1),
		http.StatusOK,
		`{"name":"Test Tenant","subnet_bits":24}`,
	},
	{
		"PATCH",
		"/tenants/093ae09b-f653-464e-9ae6-5ae28bd03a22",
		`{"name":"Updated Test Tenant","subnet_bits":4}`,
		fmt.Sprintf("application/%s", "merge-patch+json"),
		http.StatusNoContent,
		"null",
	},
	{
		"POST",
		"/tenants",
		`{"id":"093ae09b-f653-464e-9ae6-5ae28bd03a22","config":{"name":"New Tenant","subnet_bits":4}}`,
		fmt.Sprintf("application/%s", TenantsV1),
		http.StatusCreated,
		`{"id":"093ae09b-f653-464e-9ae6-5ae28bd03a22","name":"New Tenant","links":[{"rel":"self","href":"/tenants/093ae09b-f653-464e-9ae6-5ae28bd03a22"}]}`,
	},
	{
		"DELETE",
		"/tenants/093ae09b-f653-464e-9ae6-5ae28bd03a22",
		"",
		fmt.Sprintf("application/%s", TenantsV1),
		http.StatusNoContent,
		"null",
	}, {
		"POST",
		"/images",
		`{"container_format":"bare","disk_format":"raw","name":"Ubuntu","id":"b2173dd3-7ad6-4362-baa6-a68bce3565cb","visibility":"private"}`,
		fmt.Sprintf("application/%s", ImagesV1),
		http.StatusCreated,
		`{"status":"created","container_format":"bare","min_ram":0,"updated_at":"2015-11-29T22:21:42Z","owner":"bab7d5c60cd041a0a36f7c4b6e1dd978","min_disk":0,"tags":[],"locations":[],"visibility":"private","id":"b2173dd3-7ad6-4362-baa6-a68bce3565cb","size":null,"virtual_size":null,"name":"Ubuntu","checksum":null,"created_at":"2015-11-29T22:21:42Z","disk_format":"raw","properties":null,"protected":false,"self":"/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb","file":"/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb/file","schema":"/schemas/image"}`,
	},
	{
		"GET",
		"/images",
		"",
		fmt.Sprintf("application/%s", ImagesV1),
		http.StatusOK,
		`{"images":[{"status":"created","container_format":"bare","min_ram":0,"updated_at":"2015-11-29T22:21:42Z","owner":"bab7d5c60cd041a0a36f7c4b6e1dd978","min_disk":0,"tags":[],"locations":[],"visibility":"public","id":"b2173dd3-7ad6-4362-baa6-a68bce3565cb","size":null,"virtual_size":null,"name":"Ubuntu","checksum":null,"created_at":"2015-11-29T22:21:42Z","disk_format":"raw","properties":null,"protected":false,"self":"/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb","file":"/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb/file","schema":"/schemas/image"},{"status":"created","container_format":"bare","min_ram":0,"updated_at":"2015-11-29T22:21:42Z","owner":"bab7d5c60cd041a0a36f7c4b6e1dd978","min_disk":0,"tags":[],"locations":[],"visibility":"public","id":"b2173dd3-7ad6-4362-baa6-a68bce3565cb","size":null,"virtual_size":null,"name":"Ubuntu","checksum":null,"created_at":"2015-11-29T22:21:42Z","disk_format":"raw","properties":null,"protected":false,"self":"/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb","file":"/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb/file","schema":"/schemas/image"}],"schema":"/v2/schemas/images","first":"/v2/images"}`,
	},
	{
		"GET",
		"/images/1bea47ed-f6a9-463b-b423-14b9cca9ad27",
		"",
		fmt.Sprintf("application/%s", ImagesV1),
		http.StatusOK,
		`{"status":"active","container_format":"bare","min_ram":0,"updated_at":"2014-05-05T17:15:11Z","owner":"5ef70662f8b34079a6eddb8da9d75fe8","min_disk":0,"tags":[],"locations":[],"visibility":"public","id":"1bea47ed-f6a9-463b-b423-14b9cca9ad27","size":13167616,"virtual_size":null,"name":"cirros-0.3.2-x86_64-disk","checksum":"64d7c1cd2b6f60c92c14662941cb7913","created_at":"2014-05-05T17:15:10Z","disk_format":"qcow2","properties":null,"protected":false,"self":"/images/1bea47ed-f6a9-463b-b423-14b9cca9ad27","file":"/images/1bea47ed-f6a9-463b-b423-14b9cca9ad27/file","schema":"/schemas/image"}`,
	},
	{
		"DELETE",
		"/images/1bea47ed-f6a9-463b-b423-14b9cca9ad27",
		"",
		fmt.Sprintf("application/%s", ImagesV1),
		http.StatusNoContent,
		`null`,
	},
}

type testCiaoService struct{}

func (ts testCiaoService) ListPools() ([]types.Pool, error) {
	self := types.Link{
		Rel:  "self",
		Href: "/pools/ba58f471-0735-4773-9550-188e2d012941",
	}

	resp := types.Pool{
		ID:       "ba58f471-0735-4773-9550-188e2d012941",
		Name:     "testpool",
		Free:     0,
		TotalIPs: 0,
		Subnets:  []types.ExternalSubnet{},
		IPs:      []types.ExternalIP{},
		Links:    []types.Link{self},
	}

	return []types.Pool{resp}, nil
}

func (ts testCiaoService) AddPool(name string, subnet *string, ips []string) (types.Pool, error) {
	return types.Pool{}, nil
}

func (ts testCiaoService) ShowPool(id string) (types.Pool, error) {
	fmt.Println("ShowPool")
	self := types.Link{
		Rel:  "self",
		Href: "/pools/ba58f471-0735-4773-9550-188e2d012941",
	}

	resp := types.Pool{
		ID:       "ba58f471-0735-4773-9550-188e2d012941",
		Name:     "testpool",
		Free:     0,
		TotalIPs: 0,
		Subnets:  []types.ExternalSubnet{},
		IPs:      []types.ExternalIP{},
		Links:    []types.Link{self},
	}

	return resp, nil
}

func (ts testCiaoService) DeletePool(id string) error {
	return nil
}

func (ts testCiaoService) AddAddress(poolID string, subnet *string, ips []string) error {
	return nil
}

func (ts testCiaoService) RemoveAddress(poolID string, subnet *string, extIP *string) error {
	return nil
}

func (ts testCiaoService) ListMappedAddresses(tenant *string) []types.MappedIP {
	var ref string

	m := types.MappedIP{
		ID:         "ba58f471-0735-4773-9550-188e2d012941",
		ExternalIP: "192.168.0.1",
		InternalIP: "172.16.0.1",
		TenantID:   "8a497c68-a88a-4c1c-be56-12a4883208d3",
		PoolID:     "f384ffd8-e7bd-40c2-8552-2efbe7e3ad6e",
		PoolName:   "mypool",
	}

	if tenant != nil {
		ref = fmt.Sprintf("%s/external-ips/%s", *tenant, m.ID)
	} else {
		ref = fmt.Sprintf("/external-ips/%s", m.ID)
	}

	link := types.Link{
		Rel:  "self",
		Href: ref,
	}

	m.Links = []types.Link{link}

	if tenant == nil {
		ref := fmt.Sprintf("/pools/%s", m.PoolID)

		link := types.Link{
			Rel:  "pool",
			Href: ref,
		}

		m.Links = append(m.Links, link)
	}

	return []types.MappedIP{m}
}

func (ts testCiaoService) MapAddress(tenantID string, name *string, instanceID string) error {
	return nil
}

func (ts testCiaoService) UnMapAddress(string) error {
	return nil
}

func (ts testCiaoService) CreateWorkload(req types.Workload) (types.Workload, error) {
	req.ID = "ba58f471-0735-4773-9550-188e2d012941"
	return req, nil
}

func (ts testCiaoService) DeleteWorkload(tenant string, workload string) error {
	return nil
}

func (ts testCiaoService) ShowWorkload(tenant string, ID string) (types.Workload, error) {
	return types.Workload{
		ID:          "ba58f471-0735-4773-9550-188e2d012941",
		TenantID:    tenant,
		Description: "testWorkload",
		FWType:      payloads.Legacy,
		VMType:      payloads.QEMU,
		Config:      "this will totally work!",
	}, nil
}

func (ts testCiaoService) ListWorkloads(tenant string) ([]types.Workload, error) {
	return []types.Workload{
		{
			ID:          "ba58f471-0735-4773-9550-188e2d012941",
			TenantID:    tenant,
			Description: "testWorkload",
			FWType:      payloads.Legacy,
			VMType:      payloads.QEMU,
			Config:      "this will totally work!",
		},
	}, nil
}

func (ts testCiaoService) ListQuotas(tenantID string) []types.QuotaDetails {
	return []types.QuotaDetails{
		{Name: "test-quota-1", Value: 10, Usage: 3},
		{Name: "test-quota-2", Value: -1, Usage: 10},
		{Name: "test-limit", Value: 123, Usage: 0},
	}
}

func (ts testCiaoService) EvacuateNode(nodeID string) error {
	return nil
}

func (ts testCiaoService) RestoreNode(nodeID string) error {
	return nil
}

func (ts testCiaoService) UpdateQuotas(tenantID string, qds []types.QuotaDetails) error {
	return nil
}

func (ts testCiaoService) ListTenants() ([]types.TenantSummary, error) {
	summary := types.TenantSummary{
		ID:   "bc70dcd6-7298-4933-98a9-cded2d232d02",
		Name: "Test Tenant",
	}

	ref := fmt.Sprintf("/tenants/%s", summary.ID)

	link := types.Link{
		Rel:  "self",
		Href: ref,
	}

	summary.Links = append(summary.Links, link)

	return []types.TenantSummary{summary}, nil
}

func (ts testCiaoService) ShowTenant(ID string) (types.TenantConfig, error) {
	config := types.TenantConfig{
		Name:       "Test Tenant",
		SubnetBits: 24,
	}

	return config, nil
}

func (ts testCiaoService) PatchTenant(string, []byte) error {
	return nil
}

func (ts testCiaoService) CreateTenant(ID string, config types.TenantConfig) (types.TenantSummary, error) {
	summary := types.TenantSummary{
		ID:   ID,
		Name: config.Name,
	}

	ref := fmt.Sprintf("/tenants/%s", summary.ID)
	link := types.Link{
		Rel:  "self",
		Href: ref,
	}
	summary.Links = append(summary.Links, link)

	return summary, nil
}

func (ts testCiaoService) DeleteTenant(string) error {
	return nil
}

func (ts testCiaoService) CreateImage(tenantID string, req CreateImageRequest) (DefaultResponse, error) {
	format := Bare
	name := "Ubuntu"
	createdAt, _ := time.Parse(time.RFC3339, "2015-11-29T22:21:42Z")
	updatedAt, _ := time.Parse(time.RFC3339, "2015-11-29T22:21:42Z")
	minDisk := 0
	minRAM := 0
	owner := "bab7d5c60cd041a0a36f7c4b6e1dd978"

	return DefaultResponse{
		Status:          types.Created,
		ContainerFormat: &format,
		CreatedAt:       createdAt,
		Tags:            make([]string, 0),
		DiskFormat:      Raw,
		Visibility:      types.Private,
		UpdatedAt:       &updatedAt,
		Locations:       make([]string, 0),
		Self:            "/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb",
		MinDisk:         &minDisk,
		Protected:       false,
		ID:              "b2173dd3-7ad6-4362-baa6-a68bce3565cb",
		File:            "/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb/file",
		Owner:           &owner,
		MinRAM:          &minRAM,
		Schema:          "/schemas/image",
		Name:            &name,
	}, nil
}

func (ts testCiaoService) ListImages(tenantID string) ([]DefaultResponse, error) {
	format := Bare
	name := "Ubuntu"
	createdAt, _ := time.Parse(time.RFC3339, "2015-11-29T22:21:42Z")
	updatedAt, _ := time.Parse(time.RFC3339, "2015-11-29T22:21:42Z")
	minDisk := 0
	minRAM := 0
	owner := "bab7d5c60cd041a0a36f7c4b6e1dd978"

	image := DefaultResponse{
		Status:          types.Created,
		ContainerFormat: &format,
		CreatedAt:       createdAt,
		Tags:            make([]string, 0),
		DiskFormat:      Raw,
		Visibility:      types.Public,
		UpdatedAt:       &updatedAt,
		Locations:       make([]string, 0),
		Self:            "/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb",
		MinDisk:         &minDisk,
		Protected:       false,
		ID:              "b2173dd3-7ad6-4362-baa6-a68bce3565cb",
		File:            "/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb/file",
		Owner:           &owner,
		MinRAM:          &minRAM,
		Schema:          "/schemas/image",
		Name:            &name,
	}

	var images []DefaultResponse

	if tenantID == string(image.Visibility) {
		images = append(images, image)
	}

	return images, nil
}

func (ts testCiaoService) GetImage(tenantID, ID string) (DefaultResponse, error) {
	imageID := "1bea47ed-f6a9-463b-b423-14b9cca9ad27"
	format := Bare
	name := "cirros-0.3.2-x86_64-disk"
	createdAt, _ := time.Parse(time.RFC3339, "2014-05-05T17:15:10Z")
	updatedAt, _ := time.Parse(time.RFC3339, "2014-05-05T17:15:11Z")
	minDisk := 0
	minRAM := 0
	owner := "5ef70662f8b34079a6eddb8da9d75fe8"
	checksum := "64d7c1cd2b6f60c92c14662941cb7913"
	size := 13167616

	return DefaultResponse{
		Status:          types.Active,
		ContainerFormat: &format,
		CreatedAt:       createdAt,
		Tags:            make([]string, 0),
		DiskFormat:      QCow,
		Visibility:      types.Public,
		UpdatedAt:       &updatedAt,
		Locations:       make([]string, 0),
		Self:            fmt.Sprintf("/images/%s", imageID),
		MinDisk:         &minDisk,
		Protected:       false,
		CheckSum:        &checksum,
		ID:              imageID,

		File:   fmt.Sprintf("/images/%s/file", imageID),
		Owner:  &owner,
		MinRAM: &minRAM,
		Schema: "/schemas/image",
		Name:   &name,
		Size:   &size,
	}, nil
}

func (ts testCiaoService) UploadImage(string, string, io.Reader) (NoContentImageResponse, error) {
	return NoContentImageResponse{}, nil
}

func (ts testCiaoService) DeleteImage(string, string) (NoContentImageResponse, error) {
	return NoContentImageResponse{}, nil
}

func TestResponse(t *testing.T) {
	var ts testCiaoService

	mux := Routes(Config{"", ts}, nil)

	for i, tt := range tests {
		req, err := http.NewRequest(tt.method, tt.request, bytes.NewBuffer([]byte(tt.requestBody)))
		if err != nil {
			t.Fatal(err)
		}

		req = req.WithContext(service.SetPrivilege(req.Context(), true))

		rr := httptest.NewRecorder()
		req.Header.Set("Content-Type", tt.media)

		mux.ServeHTTP(rr, req)

		status := rr.Code
		if status != tt.expectedStatus {
			t.Errorf("test %d: got %v, expected %v", i, status, tt.expectedStatus)
		}

		if rr.Body.String() != tt.expectedResponse {
			t.Errorf("test %d: %s: failed\ngot: %v\nexp: %v", i, tt.request, rr.Body.String(), tt.expectedResponse)
		}
	}
}

func TestRoutes(t *testing.T) {
	var ts testCiaoService
	config := Config{"", ts}

	r := Routes(config, nil)
	if r == nil {
		t.Fatalf("No routes returned")
	}
}
