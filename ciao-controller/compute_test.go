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
	"github.com/01org/ciao/ssntp"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func testHTTPRequest(t *testing.T, method string, URL string, expectedResponse int, data []byte) []byte {
	req, err := http.NewRequest(method, URL, bytes.NewBuffer(data))
	req.Header.Set("X-Auth-Token", "imavalidtoken")
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedResponse {
		var msg string

		body, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			msg = string(body)
		}

		t.Fatalf("expected: %d, got: %d, msg: %s", expectedResponse, resp.StatusCode, msg)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	return body
}

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

	body := testHTTPRequest(t, "POST", url, http.StatusAccepted, b)

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

	body := testHTTPRequest(t, "GET", url, http.StatusOK, nil)

	var s payloads.ComputeServers
	err := json.Unmarshal(body, &s)
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

	body := testHTTPRequest(t, "GET", url, http.StatusOK, nil)

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
		url := tURL + s1.ID

		body := testHTTPRequest(t, "GET", url, http.StatusOK, nil)

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

func TestDeleteServer(t *testing.T) {
	tenant, err := context.ds.GetTenant(computeTestUser)
	if err != nil {
		t.Fatal(err)
	}

	// instances have to be assigned to a node to be deleted
	client := newTestClient(0, ssntp.AGENT)
	defer client.ssntp.Close()

	tURL := computeURL + "/v2.1/" + tenant.ID + "/servers/"

	servers := testCreateServer(t, 10)
	if servers.TotalServers != 10 {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	client.sendStats()

	s := testListServerDetailsTenant(t, tenant.ID)

	if s.TotalServers < 1 {
		t.Fatal("Not enough servers returned")
	}

	for _, s1 := range s.Servers {
		url := tURL + s1.ID
		if s1.HostID != "" {
			_ = testHTTPRequest(t, "DELETE", url, http.StatusAccepted, nil)
		} else {
			_ = testHTTPRequest(t, "DELETE", url, http.StatusInternalServerError, nil)
		}

	}
}

func TestServersActionStart(t *testing.T) {
	tenant, err := context.ds.GetTenant(computeTestUser)
	if err != nil {
		t.Fatal(err)
	}

	url := computeURL + "/v2.1/" + tenant.ID + "/servers/action"

	client := newTestClient(0, ssntp.AGENT)
	defer client.ssntp.Close()

	servers := testCreateServer(t, 1)
	if servers.TotalServers != 1 {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	client.sendStats()

	time.Sleep(1 * time.Second)

	err = context.stopInstance(servers.Servers[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)
	client.sendStats()

	var ids []string
	ids = append(ids, servers.Servers[0].ID)

	cmd := payloads.CiaoServersAction{
		Action:    "os-start",
		ServerIDs: ids,
	}

	b, err := json.Marshal(cmd)
	if err != nil {
		t.Fatal(err)
	}

	_ = testHTTPRequest(t, "POST", url, http.StatusAccepted, b)
}

func TestServersActionStop(t *testing.T) {
	tenant, err := context.ds.GetTenant(computeTestUser)
	if err != nil {
		t.Fatal(err)
	}

	url := computeURL + "/v2.1/" + tenant.ID + "/servers/action"

	client := newTestClient(0, ssntp.AGENT)
	defer client.ssntp.Close()

	servers := testCreateServer(t, 1)
	if servers.TotalServers != 1 {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	client.sendStats()

	time.Sleep(1 * time.Second)

	var ids []string
	ids = append(ids, servers.Servers[0].ID)

	cmd := payloads.CiaoServersAction{
		Action:    "os-stop",
		ServerIDs: ids,
	}

	b, err := json.Marshal(cmd)
	if err != nil {
		t.Fatal(err)
	}

	_ = testHTTPRequest(t, "POST", url, http.StatusAccepted, b)
}

func TestListFlavors(t *testing.T) {
	tenant, err := context.ds.GetTenant(computeTestUser)
	if err != nil {
		t.Fatal(err)
	}

	url := computeURL + "/v2.1/" + tenant.ID + "/flavors"

	wls, err := context.ds.GetWorkloads()
	if err != nil {
		t.Fatal(err)
	}

	body := testHTTPRequest(t, "GET", url, http.StatusOK, nil)

	var flavors payloads.ComputeFlavors
	err = json.Unmarshal(body, &flavors)
	if err != nil {
		t.Fatal(err)
	}

	if len(flavors.Flavors) != len(wls) {
		t.Fatal("Incorrect number of flavors returned")
	}

	var matched int

	for _, f := range flavors.Flavors {
		for _, w := range wls {
			if w.ID == f.ID && w.Description == f.Name {
				matched++
			}
		}
	}

	if matched != len(wls) {
		t.Fatal("Flavor information didn't match workload information")
	}
}
