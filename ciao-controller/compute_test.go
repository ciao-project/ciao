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
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/01org/ciao/testutil"
)

func testHTTPRequest(t *testing.T, method string, URL string, expectedResponse int, data []byte, validToken bool) []byte {
	req, err := http.NewRequest(method, URL, bytes.NewBuffer(data))
	if err != nil {
		t.Fatal(err)
	}

	if validToken {
		req.Header.Set("X-Auth-Token", "imavalidtoken")
	}
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
	tenant, err := context.ds.GetTenant(testutil.ComputeUser)
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

	url := testutil.ComputeURL + "/v2.1/" + tenant.ID + "/servers"

	var server payloads.ComputeCreateServer
	server.Server.MaxInstances = n
	server.Server.Workload = wls[0].ID

	b, err := json.Marshal(server)
	if err != nil {
		t.Fatal(err)
	}

	body := testHTTPRequest(t, "POST", url, http.StatusAccepted, b, true)

	servers := payloads.NewComputeServers()

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
	url := testutil.ComputeURL + "/v2.1/" + tenantID + "/servers/detail"

	body := testHTTPRequest(t, "GET", url, http.StatusOK, nil, true)

	s := payloads.NewComputeServers()
	err := json.Unmarshal(body, &s)
	if err != nil {
		t.Fatal(err)
	}

	return s
}

func TestCreateSingleServer(t *testing.T) {
	_ = testCreateServer(t, 1)
}

func TestCreateSingleServerInvalidToken(t *testing.T) {
	tenant, err := context.ds.GetTenant(testutil.ComputeUser)
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

	url := testutil.ComputeURL + "/v2.1/" + tenant.ID + "/servers"

	var server payloads.ComputeCreateServer
	server.Server.MaxInstances = 1
	server.Server.Workload = wls[0].ID

	b, err := json.Marshal(server)
	if err != nil {
		t.Fatal(err)
	}

	_ = testHTTPRequest(t, "POST", url, http.StatusUnauthorized, b, false)
}

func TestListServerDetailsTenant(t *testing.T) {
	tenant, err := context.ds.GetTenant(testutil.ComputeUser)
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

func TestListServerDetailsTenantInvalidToken(t *testing.T) {
	tenant, err := context.ds.GetTenant(testutil.ComputeUser)
	if err != nil {
		t.Fatal(err)
	}

	servers := testCreateServer(t, 1)
	if servers.TotalServers != 1 {
		t.Fatal(err)
	}

	url := testutil.ComputeURL + "/v2.1/" + tenant.ID + "/servers/detail"
	_ = testHTTPRequest(t, "GET", url, http.StatusUnauthorized, nil, false)
}

func testListServerDetailsWorkload(t *testing.T, httpExpectedStatus int, validToken bool) {
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

	url := testutil.ComputeURL + "/v2.1/flavors/" + wls[0].ID + "/servers/detail"

	body := testHTTPRequest(t, "GET", url, httpExpectedStatus, nil, validToken)

	var s payloads.ComputeServers
	err = json.Unmarshal(body, &s)
	if err != nil {
		t.Fatal(err)
	}

	if s.TotalServers < 10 {
		t.Fatal("Did not return correct number of servers")
	}
}

func TestListServerDetailsWorkload(t *testing.T) {
	testListServerDetailsWorkload(t, http.StatusOK, true)
}

func testShowServerDetails(t *testing.T, httpExpectedStatus int, validToken bool) {
	tenant, err := context.ds.GetTenant(testutil.ComputeUser)
	if err != nil {
		t.Fatal(err)
	}

	tURL := testutil.ComputeURL + "/v2.1/" + tenant.ID + "/servers/"

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

		body := testHTTPRequest(t, "GET", url, httpExpectedStatus, nil, validToken)
		// stop evaluating in case the scenario is InvalidToken
		if httpExpectedStatus == 401 {
			return
		}

		var s2 payloads.ComputeServer
		err = json.Unmarshal(body, &s2)
		if err != nil {
			t.Fatal(err)
		}

		if reflect.DeepEqual(s1, s2.Server) == false {
			t.Fatal("Server details not correct")
			//t.Fatalf("Server details not correct %s %s", s1, s2.Server)
		}
	}
}

func TestShowServerDetails(t *testing.T) {
	testShowServerDetails(t, http.StatusOK, true)
}

func TestShowServerDetailsInvalidToken(t *testing.T) {
	testShowServerDetails(t, http.StatusUnauthorized, false)
}

func testDeleteServer(t *testing.T, httpExpectedStatus int, httpExpectedErrorStatus int, validToken bool) {
	tenant, err := context.ds.GetTenant(testutil.ComputeUser)
	if err != nil {
		t.Fatal(err)
	}

	// instances have to be assigned to a node to be deleted
	client, err := testutil.NewSsntpTestClientConnection("DeleteServer", ssntp.AGENT, testutil.AgentUUID)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Shutdown()

	tURL := testutil.ComputeURL + "/v2.1/" + tenant.ID + "/servers/"

	servers := testCreateServer(t, 10)
	if servers.TotalServers != 10 {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	sendStatsCmd(client, t)

	time.Sleep(2 * time.Second)

	s := testListServerDetailsTenant(t, tenant.ID)

	if s.TotalServers < 1 {
		t.Fatal("Not enough servers returned")
	}

	for _, s1 := range s.Servers {
		url := tURL + s1.ID
		if s1.HostID != "" {
			_ = testHTTPRequest(t, "DELETE", url, httpExpectedStatus, nil, validToken)
		} else {
			_ = testHTTPRequest(t, "DELETE", url, httpExpectedErrorStatus, nil, validToken)
		}
	}
}

func TestDeleteServer(t *testing.T) {
	testDeleteServer(t, http.StatusAccepted, http.StatusInternalServerError, true)
}

func TestDeleteServerInvalidToken(t *testing.T) {
	testDeleteServer(t, http.StatusUnauthorized, http.StatusUnauthorized, false)
}

func testServersActionStart(t *testing.T, httpExpectedStatus int, validToken bool) {
	tenant, err := context.ds.GetTenant(testutil.ComputeUser)
	if err != nil {
		t.Fatal(err)
	}

	url := testutil.ComputeURL + "/v2.1/" + tenant.ID + "/servers/action"

	client, err := testutil.NewSsntpTestClientConnection("ServersActionStart", ssntp.AGENT, testutil.AgentUUID)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Shutdown()

	servers := testCreateServer(t, 1)
	if servers.TotalServers != 1 {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	sendStatsCmd(client, t)

	time.Sleep(1 * time.Second)

	err = context.stopInstance(servers.Servers[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)

	sendStatsCmd(client, t)

	time.Sleep(1 * time.Second)

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

	_ = testHTTPRequest(t, "POST", url, httpExpectedStatus, b, validToken)
}

func TestServersActionStart(t *testing.T) {
	testServersActionStart(t, http.StatusAccepted, true)
}

func TestServersActionStartInvalidToken(t *testing.T) {
	testServersActionStart(t, http.StatusUnauthorized, false)
}

func testServersActionStop(t *testing.T, httpExpectedStatus int, action string) {
	tenant, err := context.ds.GetTenant(testutil.ComputeUser)
	if err != nil {
		t.Fatal(err)
	}

	url := testutil.ComputeURL + "/v2.1/" + tenant.ID + "/servers/action"

	client, err := testutil.NewSsntpTestClientConnection("ServersActionStop", ssntp.AGENT, testutil.AgentUUID)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Shutdown()

	servers := testCreateServer(t, 1)
	if servers.TotalServers != 1 {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	sendStatsCmd(client, t)

	time.Sleep(1 * time.Second)

	var ids []string
	ids = append(ids, servers.Servers[0].ID)

	cmd := payloads.CiaoServersAction{
		Action:    action,
		ServerIDs: ids,
	}

	b, err := json.Marshal(cmd)
	if err != nil {
		t.Fatal(err)
	}

	_ = testHTTPRequest(t, "POST", url, httpExpectedStatus, b, true)
}

func TestServersActionStop(t *testing.T) {
	testServersActionStop(t, http.StatusAccepted, "os-stop")
}

func TestServersActionStopWrongAction(t *testing.T) {
	testServersActionStop(t, http.StatusServiceUnavailable, "wrong-action")
}

func testServerActionStop(t *testing.T, httpExpectedStatus int, validToken bool) {
	action := "os-stop"

	tenant, err := context.ds.GetTenant(testutil.ComputeUser)
	if err != nil {
		t.Fatal(err)
	}

	client, err := testutil.NewSsntpTestClientConnection("ServerActionStop", ssntp.AGENT, testutil.AgentUUID)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Shutdown()

	servers := testCreateServer(t, 1)
	if servers.TotalServers != 1 {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	sendStatsCmd(client, t)

	time.Sleep(1 * time.Second)

	url := testutil.ComputeURL + "/v2.1/" + tenant.ID + "/servers/" + servers.Servers[0].ID + "/action"
	_ = testHTTPRequest(t, "POST", url, httpExpectedStatus, []byte(action), validToken)
}

func TestServerActionStop(t *testing.T) {
	testServerActionStop(t, http.StatusAccepted, true)
}

func TestServerActionStopInvalidToken(t *testing.T) {
	testServerActionStop(t, http.StatusUnauthorized, false)
}

func TestServerActionStart(t *testing.T) {
	action := "os-start"

	tenant, err := context.ds.GetTenant(testutil.ComputeUser)
	if err != nil {
		t.Fatal(err)
	}

	client, err := testutil.NewSsntpTestClientConnection("ServerActionStart", ssntp.AGENT, testutil.AgentUUID)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Shutdown()

	servers := testCreateServer(t, 1)
	if servers.TotalServers != 1 {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)

	sendStatsCmd(client, t)

	time.Sleep(1 * time.Second)

	serverCh := server.AddCmdChan(ssntp.STOP)

	err = context.stopInstance(servers.Servers[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = server.GetCmdChanResult(serverCh, ssntp.STOP)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)

	sendStatsCmd(client, t)

	time.Sleep(1 * time.Second)

	url := testutil.ComputeURL + "/v2.1/" + tenant.ID + "/servers/" + servers.Servers[0].ID + "/action"
	_ = testHTTPRequest(t, "POST", url, http.StatusAccepted, []byte(action), true)
}

func testListFlavors(t *testing.T, httpExpectedStatus int, data []byte, validToken bool) {
	tenant, err := context.ds.GetTenant(testutil.ComputeUser)
	if err != nil {
		t.Fatal(err)
	}

	url := testutil.ComputeURL + "/v2.1/" + tenant.ID + "/flavors"

	wls, err := context.ds.GetWorkloads()
	if err != nil {
		t.Fatal(err)
	}

	body := testHTTPRequest(t, "GET", url, httpExpectedStatus, data, validToken)
	// stop evaluating in case the scenario is InvalidToken
	if httpExpectedStatus == 401 {
		return
	}

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

func TestListFlavors(t *testing.T) {
	testListFlavors(t, http.StatusOK, nil, true)
}

func TestListFlavorsInvalidToken(t *testing.T) {
	testListFlavors(t, http.StatusUnauthorized, nil, false)
}

func testShowFlavorDetails(t *testing.T, httpExpectedStatus int, validToken bool) {
	tenant, err := context.ds.GetTenant(testutil.ComputeUser)
	if err != nil {
		t.Fatal(err)
	}

	tURL := testutil.ComputeURL + "/v2.1/" + tenant.ID + "/flavors/"

	wls, err := context.ds.GetWorkloads()
	if err != nil {
		t.Fatal(err)
	}

	for _, w := range wls {
		details := payloads.FlavorDetails{
			OsFlavorAccessIsPublic: true,
			ID:   w.ID,
			Disk: w.ImageID,
			Name: w.Description,
		}

		defaults := w.Defaults
		for r := range defaults {
			switch defaults[r].Type {
			case payloads.VCPUs:
				details.Vcpus = defaults[r].Value
			case payloads.MemMB:
				details.RAM = defaults[r].Value
			}
		}

		url := tURL + w.ID
		body := testHTTPRequest(t, "GET", url, httpExpectedStatus, nil, validToken)
		// stop evaluating in case the scenario is InvalidToken
		if httpExpectedStatus == 401 {
			return
		}

		var f payloads.ComputeFlavorDetails

		err = json.Unmarshal(body, &f)
		if err != nil {
			t.Fatal(err)
		}

		if reflect.DeepEqual(details, f.Flavor) == false {
			t.Fatal("Flavor details not correct")
		}
	}
}

func TestShowFlavorDetails(t *testing.T) {
	testShowFlavorDetails(t, http.StatusOK, true)
}

func TestShowFlavorDetailsInvalidToken(t *testing.T) {
	testShowFlavorDetails(t, http.StatusUnauthorized, false)
}

func testListFlavorsDetails(t *testing.T, httpExpectedStatus int, data []byte, validToken bool) {
	tenant, err := context.ds.GetTenant(testutil.ComputeUser)
	if err != nil {
		t.Fatal(err)
	}

	url := testutil.ComputeURL + "/v2.1/" + tenant.ID + "/flavors/detail"
	_ = testHTTPRequest(t, "GET", url, httpExpectedStatus, data, validToken)
}

func TestListFlavorsDetails(t *testing.T) {
	testListFlavorsDetails(t, http.StatusOK, nil, true)
}

func TestListFlavorsDetailsInvalidToken(t *testing.T) {
	testListFlavorsDetails(t, http.StatusUnauthorized, nil, false)
}

func testListTenantResources(t *testing.T, httpExpectedStatus int, validToken bool) {
	var usage payloads.CiaoUsageHistory

	endTime := time.Now()
	startTime := endTime.Add(-15 * time.Minute)

	tenant, err := context.ds.GetTenant(testutil.ComputeUser)
	if err != nil {
		t.Fatal(err)
	}

	tURL := testutil.ComputeURL + "/v2.1/" + tenant.ID + "/resources?"

	usage.Usages, err = context.ds.GetTenantUsage(tenant.ID, startTime, endTime)
	if err != nil {
		t.Fatal(err)
	}

	v := url.Values{}
	v.Add("start_date", startTime.Format(time.RFC3339))
	v.Add("end_date", endTime.Format(time.RFC3339))

	tURL += v.Encode()

	body := testHTTPRequest(t, "GET", tURL, httpExpectedStatus, nil, validToken)
	// stop evaluating in case the scenario is InvalidToken
	if httpExpectedStatus == 401 {
		return
	}

	var result payloads.CiaoUsageHistory

	err = json.Unmarshal(body, &result)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(usage, result) == false {
		t.Fatal("Tenant usage not correct")
	}
}

func TestListTenantResources(t *testing.T) {
	testListTenantResources(t, http.StatusOK, true)
}

func TestListTenantResourcesInvalidToken(t *testing.T) {
	testListTenantResources(t, http.StatusUnauthorized, false)
}

func testListTenantQuotas(t *testing.T, httpExpectedStatus int, validToken bool) {
	tenant, err := context.ds.GetTenant(testutil.ComputeUser)
	if err != nil {
		t.Fatal(err)
	}

	url := testutil.ComputeURL + "/v2.1/" + tenant.ID + "/quotas"

	var expected payloads.CiaoTenantResources

	for _, resource := range tenant.Resources {
		switch resource.Rtype {
		case instances:
			expected.InstanceLimit = resource.Limit
			expected.InstanceUsage = resource.Usage

		case vcpu:
			expected.VCPULimit = resource.Limit
			expected.VCPUUsage = resource.Usage

		case memory:
			expected.MemLimit = resource.Limit
			expected.MemUsage = resource.Usage

		case disk:
			expected.DiskLimit = resource.Limit
			expected.DiskUsage = resource.Usage
		}
	}

	expected.ID = tenant.ID

	body := testHTTPRequest(t, "GET", url, httpExpectedStatus, nil, validToken)
	// stop evaluating in case the scenario is InvalidToken
	if httpExpectedStatus == 401 {
		return
	}

	var result payloads.CiaoTenantResources

	err = json.Unmarshal(body, &result)
	if err != nil {
		t.Fatal(err)
	}

	expected.Timestamp = result.Timestamp

	if reflect.DeepEqual(expected, result) == false {
		t.Fatal("Tenant quotas not correct")
	}
}

func TestListTenantQuotas(t *testing.T) {
	testListTenantQuotas(t, http.StatusOK, true)
}

func TestListTenantQuotasInvalidToken(t *testing.T) {
	testListTenantQuotas(t, http.StatusUnauthorized, false)
}

func testListEventsTenant(t *testing.T, httpExpectedStatus int, validToken bool) {
	tenant, err := context.ds.GetTenant(testutil.ComputeUser)
	if err != nil {
		t.Fatal(err)
	}

	url := testutil.ComputeURL + "/v2.1/" + tenant.ID + "/events"

	expected := payloads.NewCiaoEvents()

	logs, err := context.ds.GetEventLog()
	if err != nil {
		t.Fatal(err)
	}

	for _, l := range logs {
		if tenant.ID != l.TenantID {
			continue
		}

		event := payloads.CiaoEvent{
			Timestamp: l.Timestamp,
			TenantID:  l.TenantID,
			EventType: l.EventType,
			Message:   l.Message,
		}
		expected.Events = append(expected.Events, event)
	}

	body := testHTTPRequest(t, "GET", url, httpExpectedStatus, nil, validToken)

	var result payloads.CiaoEvents

	err = json.Unmarshal(body, &result)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(expected, result) == false {
		t.Fatal("Tenant events not correct")
	}
}

func TestListEventsTenant(t *testing.T) {
	testListEventsTenant(t, http.StatusOK, true)
}

func testListNodeServers(t *testing.T, httpExpectedStatus int, validToken bool) {
	computeNodes := context.ds.GetNodeLastStats()

	for _, n := range computeNodes.Nodes {
		instances, err := context.ds.GetAllInstancesByNode(n.ID)
		if err != nil {
			t.Fatal(err)
		}

		url := testutil.ComputeURL + "/v2.1/nodes/" + n.ID + "/servers/detail"

		body := testHTTPRequest(t, "GET", url, httpExpectedStatus, nil, validToken)
		// stop evaluating in case the scenario is InvalidToken
		if httpExpectedStatus == 401 {
			return
		}

		var result payloads.CiaoServersStats

		err = json.Unmarshal(body, &result)
		if err != nil {
			t.Fatal(err)
		}

		if result.TotalServers != len(instances) {
			t.Fatal("Incorrect number of servers")
		}

		// TBD: make sure result exactly matches expected results.
		// this isn't done now because the list of instances is
		// possibly out of order
	}
}

func TestListNodeServers(t *testing.T) {
	testListNodeServers(t, http.StatusOK, true)
}

func TestListNodeServersInvalidToken(t *testing.T) {
	testListNodeServers(t, http.StatusUnauthorized, false)
}

func testListTenants(t *testing.T, httpExpectedStatus int, validToken bool) {
	tenants, err := context.ds.GetAllTenants()
	if err != nil {
		t.Fatal(err)
	}

	expected := payloads.NewCiaoComputeTenants()

	for _, tenant := range tenants {
		expected.Tenants = append(expected.Tenants,
			struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{
				ID:   tenant.ID,
				Name: tenant.Name,
			},
		)
	}

	url := testutil.ComputeURL + "/v2.1/tenants"

	body := testHTTPRequest(t, "GET", url, httpExpectedStatus, nil, validToken)
	// stop evaluating in case the scenario is InvalidToken
	if httpExpectedStatus == 401 {
		return
	}

	var result payloads.CiaoComputeTenants

	err = json.Unmarshal(body, &result)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(expected, result) == false {
		t.Fatal("Tenant list not correct")
	}
}

func TestListTenants(t *testing.T) {
	testListTenants(t, http.StatusOK, true)
}

func TestListTenantsInvalidToken(t *testing.T) {
	testListTenants(t, http.StatusUnauthorized, false)
}

func testListNodes(t *testing.T, httpExpectedStatus int, validToken bool) {
	expected := context.ds.GetNodeLastStats()

	summary, err := context.ds.GetNodeSummary()
	if err != nil {
		t.Fatal(err)
	}

	for _, node := range summary {
		for i := range expected.Nodes {
			if expected.Nodes[i].ID != node.NodeID {
				continue
			}

			expected.Nodes[i].TotalInstances = node.TotalInstances
			expected.Nodes[i].TotalRunningInstances = node.TotalRunningInstances
			expected.Nodes[i].TotalPendingInstances = node.TotalPendingInstances
			expected.Nodes[i].TotalPausedInstances = node.TotalPausedInstances
			expected.Nodes[i].Timestamp = time.Time{}
		}
	}

	sort.Sort(types.SortedComputeNodesByID(expected.Nodes))

	url := testutil.ComputeURL + "/v2.1/nodes"

	body := testHTTPRequest(t, "GET", url, httpExpectedStatus, nil, validToken)
	// stop evaluating in case the scenario is InvalidToken
	if httpExpectedStatus == 401 {
		return
	}

	var result payloads.CiaoComputeNodes

	err = json.Unmarshal(body, &result)
	if err != nil {
		t.Fatal(err)
	}

	for i := range result.Nodes {
		result.Nodes[i].Timestamp = time.Time{}
	}

	if reflect.DeepEqual(expected.Nodes, result.Nodes) == false {
		t.Fatalf("expected: \n%+v\n result: \n%+v\n", expected, result)
	}
}

func TestListNodes(t *testing.T) {
	testListNodes(t, http.StatusOK, true)
}

func TestListNodesInvalidToken(t *testing.T) {
	testListNodes(t, http.StatusUnauthorized, false)
}

func testNodeSummary(t *testing.T, httpExpectedStatus int, validToken bool) {
	var expected payloads.CiaoClusterStatus

	computeNodes := context.ds.GetNodeLastStats()

	expected.Status.TotalNodes = len(computeNodes.Nodes)
	for _, node := range computeNodes.Nodes {
		if node.Status == ssntp.READY.String() {
			expected.Status.TotalNodesReady++
		} else if node.Status == ssntp.FULL.String() {
			expected.Status.TotalNodesFull++
		} else if node.Status == ssntp.OFFLINE.String() {
			expected.Status.TotalNodesOffline++
		} else if node.Status == ssntp.MAINTENANCE.String() {
			expected.Status.TotalNodesMaintenance++
		}
	}

	url := testutil.ComputeURL + "/v2.1/nodes/summary"

	body := testHTTPRequest(t, "GET", url, httpExpectedStatus, nil, validToken)
	// stop evaluating in case the scenario is InvalidToken
	if httpExpectedStatus == 401 {
		return
	}

	var result payloads.CiaoClusterStatus

	err := json.Unmarshal(body, &result)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(expected, result) == false {
		t.Fatalf("expected: \n%+v\n result: \n%+v\n", expected, result)
	}
}

func TestNodeSummary(t *testing.T) {
	testNodeSummary(t, http.StatusOK, true)
}

func TestNodeSummaryInvalidToken(t *testing.T) {
	testNodeSummary(t, http.StatusUnauthorized, false)
}

func testListCNCIs(t *testing.T, httpExpectedStatus int, validToken bool) {
	var expected payloads.CiaoCNCIs

	cncis, err := context.ds.GetTenantCNCISummary("")
	if err != nil {
		t.Fatal(err)
	}

	var subnets []payloads.CiaoCNCISubnet

	for _, cnci := range cncis {
		if cnci.InstanceID == "" {
			continue
		}

		for _, subnet := range cnci.Subnets {
			subnets = append(subnets,
				payloads.CiaoCNCISubnet{
					Subnet: subnet,
				},
			)
		}

		expected.CNCIs = append(expected.CNCIs,
			payloads.CiaoCNCI{
				ID:       cnci.InstanceID,
				TenantID: cnci.TenantID,
				IPv4:     cnci.IPAddress,
				Subnets:  subnets,
			},
		)
	}

	url := testutil.ComputeURL + "/v2.1/cncis"

	body := testHTTPRequest(t, "GET", url, httpExpectedStatus, nil, validToken)
	// stop evaluating in case the scenario is InvalidToken
	if httpExpectedStatus == 401 {
		return
	}

	var result payloads.CiaoCNCIs

	err = json.Unmarshal(body, &result)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(expected, result) == false {
		t.Fatalf("expected: \n%+v\n result: \n%+v\n", expected, result)
	}
}

func TestListCNCIs(t *testing.T) {
	testListCNCIs(t, http.StatusOK, true)
}

func TestListCNCIsInvalidToken(t *testing.T) {
	testListCNCIs(t, http.StatusUnauthorized, false)
}

func testListCNCIDetails(t *testing.T, httpExpectedStatus int, validToken bool) {
	cncis, err := context.ds.GetTenantCNCISummary("")
	if err != nil {
		t.Fatal(err)
	}

	for _, cnci := range cncis {
		var expected payloads.CiaoCNCI

		cncis, err := context.ds.GetTenantCNCISummary(cnci.InstanceID)
		if err != nil {
			t.Fatal(err)
		}

		if len(cncis) > 0 {
			var subnets []payloads.CiaoCNCISubnet
			cnci := cncis[0]

			for _, subnet := range cnci.Subnets {
				subnets = append(subnets,
					payloads.CiaoCNCISubnet{
						Subnet: subnet,
					},
				)
			}

			expected = payloads.CiaoCNCI{
				ID:       cnci.InstanceID,
				TenantID: cnci.TenantID,
				IPv4:     cnci.IPAddress,
				Subnets:  subnets,
			}
		}

		url := testutil.ComputeURL + "/v2.1/cncis/" + cnci.InstanceID + "/detail"

		body := testHTTPRequest(t, "GET", url, httpExpectedStatus, nil, validToken)
		// stop evaluating in case the scenario is InvalidToken
		if httpExpectedStatus == 401 {
			return
		}

		var result payloads.CiaoCNCI

		err = json.Unmarshal(body, &result)
		if err != nil {
			t.Fatal(err)
		}

		if reflect.DeepEqual(expected, result) == false {
			t.Fatalf("expected: \n%+v\n result: \n%+v\n", expected, result)
		}
	}
}

func TestListCNCIDetails(t *testing.T) {
	testListCNCIDetails(t, http.StatusOK, true)
}

func TestListCNCIDetailsInvalidToken(t *testing.T) {
	testListCNCIDetails(t, http.StatusUnauthorized, false)
}

func testListTraces(t *testing.T, httpExpectedStatus int, validToken bool) {
	var expected payloads.CiaoTracesSummary

	client := testStartTracedWorkload(t)
	defer client.Shutdown()

	sendTraceReportEvent(client, t)

	time.Sleep(2 * time.Second)

	summaries, err := context.ds.GetBatchFrameSummary()
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range summaries {
		summary := payloads.CiaoTraceSummary{
			Label:     s.BatchID,
			Instances: s.NumInstances,
		}
		expected.Summaries = append(expected.Summaries, summary)
	}

	url := testutil.ComputeURL + "/v2.1/traces"

	body := testHTTPRequest(t, "GET", url, httpExpectedStatus, nil, validToken)
	// stop evaluating in case the scenario is InvalidToken
	if httpExpectedStatus == 401 {
		return
	}

	var result payloads.CiaoTracesSummary

	err = json.Unmarshal(body, &result)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(expected, result) == false {
		t.Fatalf("expected: \n%+v\n result: \n%+v\n", expected, result)
	}
}

func TestListTraces(t *testing.T) {
	testListTraces(t, http.StatusOK, true)
}

func TestListTracesInvalidToken(t *testing.T) {
	testListTraces(t, http.StatusUnauthorized, false)
}

func testListEvents(t *testing.T, httpExpectedStatus int, validToken bool) {
	url := testutil.ComputeURL + "/v2.1/events"

	expected := payloads.NewCiaoEvents()

	logs, err := context.ds.GetEventLog()
	if err != nil {
		t.Fatal(err)
	}

	for _, l := range logs {
		event := payloads.CiaoEvent{
			Timestamp: l.Timestamp,
			TenantID:  l.TenantID,
			EventType: l.EventType,
			Message:   l.Message,
		}
		expected.Events = append(expected.Events, event)
	}

	body := testHTTPRequest(t, "GET", url, httpExpectedStatus, nil, validToken)
	// stop evaluating in case the scenario is InvalidToken
	if httpExpectedStatus == 401 {
		return
	}

	var result payloads.CiaoEvents

	err = json.Unmarshal(body, &result)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(expected, result) == false {
		t.Fatalf("expected: \n%+v\n result: \n%+v\n", expected, result)
	}
}

func TestListEvents(t *testing.T) {
	testListEvents(t, http.StatusOK, true)
}

func TestListEventsInvalidToken(t *testing.T) {
	testListEvents(t, http.StatusUnauthorized, false)
}

func testClearEvents(t *testing.T, httpExpectedStatus int, validToken bool) {
	url := testutil.ComputeURL + "/v2.1/events"

	_ = testHTTPRequest(t, "DELETE", url, httpExpectedStatus, nil, validToken)
	// stop evaluating in case the scenario is InvalidToken
	if httpExpectedStatus == 401 {
		return
	}

	logs, err := context.ds.GetEventLog()
	if err != nil {
		t.Fatal(err)
	}

	if len(logs) != 0 {
		t.Fatal("Logs not cleared")
	}
}

func TestClearEvents(t *testing.T) {
	testClearEvents(t, http.StatusAccepted, true)
}

func TestClearEventsInvalidToken(t *testing.T) {
	testClearEvents(t, http.StatusUnauthorized, false)
}

func testTraceData(t *testing.T, httpExpectedStatus int, validToken bool) {
	client := testStartTracedWorkload(t)
	defer client.Shutdown()

	sendTraceReportEvent(client, t)

	time.Sleep(2 * time.Second)

	summaries, err := context.ds.GetBatchFrameSummary()
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range summaries {
		var expected payloads.CiaoTraceData

		batchStats, err := context.ds.GetBatchFrameStatistics(s.BatchID)
		if err != nil {
			t.Fatal(err)
		}

		expected.Summary = payloads.CiaoBatchFrameStat{
			NumInstances:             batchStats[0].NumInstances,
			TotalElapsed:             batchStats[0].TotalElapsed,
			AverageElapsed:           batchStats[0].AverageElapsed,
			AverageControllerElapsed: batchStats[0].AverageControllerElapsed,
			AverageLauncherElapsed:   batchStats[0].AverageLauncherElapsed,
			AverageSchedulerElapsed:  batchStats[0].AverageSchedulerElapsed,
			VarianceController:       batchStats[0].VarianceController,
			VarianceLauncher:         batchStats[0].VarianceLauncher,
			VarianceScheduler:        batchStats[0].VarianceScheduler,
		}

		url := testutil.ComputeURL + "/v2.1/traces/" + s.BatchID

		body := testHTTPRequest(t, "GET", url, httpExpectedStatus, nil, validToken)
		// stop evaluating in case the scenario is InvalidToken
		if httpExpectedStatus == 401 {
			return
		}

		var result payloads.CiaoTraceData

		err = json.Unmarshal(body, &result)
		if err != nil {
			t.Fatal(err)
		}

		if reflect.DeepEqual(expected, result) == false {
			t.Fatalf("expected: \n%+v\n result: \n%+v\n", expected, result)
		}
	}
}

func TestTraceData(t *testing.T) {
	testTraceData(t, http.StatusOK, true)
}

func TestTraceDataInvalidToken(t *testing.T) {
	testTraceData(t, http.StatusUnauthorized, false)
}
