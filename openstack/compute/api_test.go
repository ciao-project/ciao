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

package compute

import (
	"bytes"
	"net/http"
	"net/http/httptest"
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
		"/v2.1/{tenant}/servers/",
		createServer,
		`{"server":{"name":"new-server-test","imageRef": "http://glance.openstack.example.com/images/70a599e0-31e7-49b7-b260-868f441e862b","workload_id":"http://openstack.example.com/flavors/1","metadata":{"My Server Name":"Apache1"}}}`,
		http.StatusAccepted,
		`{"server":{"id":"validServerID","name":"new-server-test","imageRef":"http://glance.openstack.example.com/images/70a599e0-31e7-49b7-b260-868f441e862b","workload_id":"http://openstack.example.com/flavors/1","max_count":0,"min_count":0,"metadata":{"My Server Name":"Apache1"}}}`,
	},
	{
		"GET",
		"/v2.1/{tenant}/servers/detail?limit=1&offset=1",
		listServerDetails,
		"",
		http.StatusOK,
		`{"total_servers":1,"servers":[]}`,
	},
	{
		"GET",
		"/v2.1/{tenant}/servers/detail",
		listServerDetails,
		"",
		http.StatusOK,
		`{"total_servers":1,"servers":[{"private_addresses":[{"addr":"192.169.0.1","mac_addr":"00:02:00:01:02:03"}],"created":"0001-01-01T00:00:00Z","workload_id":"testWorkloadUUID","node_id":"nodeUUID","id":"testUUID","name":"","volumes":null,"status":"active","tenant_id":"","ssh_ip":"","ssh_port":0}]}`},
	{
		"GET",
		"/v2.1/{tenant}/servers/{server}",
		showServerDetails,
		"",
		http.StatusOK,
		`{"server":{"private_addresses":[{"addr":"192.169.0.1","mac_addr":"00:02:00:01:02:03"}],"created":"0001-01-01T00:00:00Z","workload_id":"testWorkloadUUID","node_id":"nodeUUID","id":"","name":"","volumes":null,"status":"active","tenant_id":"","ssh_ip":"","ssh_port":0}}`,
	},
	{
		"DELETE",
		"/v2.1/{tenant}/servers/{server}",
		deleteServer,
		"",
		http.StatusNoContent,
		"null",
	},
	{
		"POST",
		"/v2.1/{tenant}/servers/{server}/action",
		serverAction,
		`{"os-start":null}`,
		http.StatusAccepted,
		"null",
	},
	{
		"POST",
		"/v2.1/{tenant}/servers/{server}/action",
		serverAction,
		`{"os-stop":null}`,
		http.StatusAccepted,
		"null",
	},
}

type testComputeService struct{}

// server interfaces
func (cs testComputeService) CreateServer(tenant string, req CreateServerRequest) (interface{}, error) {
	req.Server.ID = "validServerID"
	return req, nil
}

func (cs testComputeService) ListServersDetail(tenant string) ([]ServerDetails, error) {
	var servers []ServerDetails

	server := ServerDetails{
		NodeID:     "nodeUUID",
		ID:         "testUUID",
		TenantID:   tenant,
		WorkloadID: "testWorkloadUUID",
		Status:     "active",
		PrivateAddresses: []PrivateAddresses{
			{
				Addr:    "192.169.0.1",
				MacAddr: "00:02:00:01:02:03",
			},
		},
	}

	servers = append(servers, server)

	return servers, nil
}

func (cs testComputeService) ShowServerDetails(tenant string, server string) (Server, error) {
	s := ServerDetails{
		NodeID:     "nodeUUID",
		ID:         server,
		TenantID:   tenant,
		WorkloadID: "testWorkloadUUID",
		Status:     "active",
		PrivateAddresses: []PrivateAddresses{
			{
				Addr:    "192.169.0.1",
				MacAddr: "00:02:00:01:02:03",
			},
		},
	}

	return Server{Server: s}, nil
}

func (cs testComputeService) DeleteServer(tenant string, server string) error {
	return nil
}

func (cs testComputeService) StartServer(tenant string, server string) error {
	return nil
}

func (cs testComputeService) StopServer(tenant string, server string) error {
	return nil
}

func TestAPIResponse(t *testing.T) {
	var cs testComputeService

	// TBD: add context to test definition so it can be created per
	// endpoint with either a pass testComputeService or a failure
	// one.
	context := &Context{cs}

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
	var cs testComputeService
	config := APIConfig{cs}

	r := Routes(config, nil)
	if r == nil {
		t.Fatalf("No routes returned")
	}
}

func TestPager(t *testing.T) {
	req, err := http.NewRequest("GET", "/v2.1/{tenant}/servers/detail?limit=2&offset=2", bytes.NewBuffer([]byte("")))

	if err != nil {
		t.Fatal(err)
	}
	limit, offset, _ := pagerQueryParse(req)
	if limit != 2 {
		t.Fatalf("Invalid limit registered")
	}
	if offset != 2 {
		t.Fatalf("Invalid offset registered")
	}
}
