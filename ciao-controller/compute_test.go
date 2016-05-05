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
	"bytes"
	"encoding/json"
	"github.com/01org/ciao/payloads"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
)

func testCreateServer(t *testing.T, n int) payloads.ComputeServers {
	tenant, err := context.ds.GetTenant(computeTestUser)
	if err != nil {
		t.Fatal(err)
	}

	// get a valid workload ID
	wls, err := context.ds.GetWorkloads()
	if err != nil {
		t.Fatal(err)
	}

	if len(wls) == 0 {
		t.Fatal("No valid workloads")
	}

	url := computeURL + "/v2.1/" + tenant.ID + "/servers"

	var server payloads.ComputeCreateServer
	server.Server.MaxInstances = n
	server.Server.Workload = wls[0].ID

	b, err := json.Marshal(server)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	req.Header.Set("X-Auth-Token", "imavalidtoken")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected response code: %d, got %d", http.StatusAccepted, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	var servers payloads.ComputeServers

	err = json.Unmarshal(body, &servers)
	if err != nil {
		t.Fatal(err)
	}

	if servers.TotalServers != n {
		t.Fatal("Not enough servers returned")
	}

	return servers
}

func testListServerDetailsTenant(t *testing.T, tenantID string) payloads.ComputeServers {
	url := computeURL + "/v2.1/" + tenantID + "/servers/detail"
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Auth-Token", "imavalidtoken")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected response code: %d, got %d", http.StatusOK, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	var s payloads.ComputeServers
	err = json.Unmarshal(body, &s)
	if err != nil {
		t.Fatal(err)
	}

	return s
}

func TestCreateSingleServer(t *testing.T) {
	_ = testCreateServer(t, 1)
}

func TestListServerDetailsTenant(t *testing.T) {
	tenant, err := context.ds.GetTenant(computeTestUser)
	if err != nil {
		t.Fatal(err)
	}

	servers := testCreateServer(t, 1)
	if servers.TotalServers != 1 {
		t.Fatal(err)
	}

	s := testListServerDetailsTenant(t, tenant.ID)

	if s.TotalServers < 1 {
		t.Fatal("Not enough servers returned")
	}
}

func TestListServerDetailsWorkload(t *testing.T) {
	// get a valid workload ID
	wls, err := context.ds.GetWorkloads()
	if err != nil {
		t.Fatal(err)
	}

	if len(wls) == 0 {
		t.Fatal("No valid workloads")
	}

	servers := testCreateServer(t, 10)
	if servers.TotalServers != 10 {
		t.Fatal("failed to create enough servers")
	}

	url := computeURL + "/v2.1/flavors/" + wls[0].ID + "/servers/detail"
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Auth-Token", "imavalidtoken")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Fatal(string(body))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	var s payloads.ComputeServers
	err = json.Unmarshal(body, &s)
	if err != nil {
		t.Fatal(err)
	}

	if s.TotalServers < 10 {
		t.Fatal("Did not return correct number of servers")
	}
}

func TestShowServerDetails(t *testing.T) {
	tenant, err := context.ds.GetTenant(computeTestUser)
	if err != nil {
		t.Fatal(err)
	}

	tURL := computeURL + "/v2.1/" + tenant.ID + "/servers/"

	servers := testCreateServer(t, 1)
	if servers.TotalServers != 1 {
		t.Fatal(err)
	}

	s := testListServerDetailsTenant(t, tenant.ID)

	if s.TotalServers < 1 {
		t.Fatal("Not enough servers returned")
	}

	for _, s1 := range s.Servers {
		URL := tURL + s1.ID

		req, err := http.NewRequest("GET", URL, nil)
		req.Header.Set("X-Auth-Token", "imavalidtoken")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected response code: %d, got %d", http.StatusOK, resp.StatusCode)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		var s2 payloads.ComputeServer
		err = json.Unmarshal(body, &s2)
		if err != nil {
			t.Fatal(err)
		}

		if reflect.DeepEqual(s1, s2.Server) == false {
			t.Fatal("Server details not correct")
		}
	}
}
