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
	"testing"
)

func TestCreateServer(t *testing.T) {
	tenant, err := context.ds.GetTenant(computeTestUser)
	if err != nil {
		t.Fatal(err)
	}

	// get a valid workload ID
	wls, err := context.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		t.Fatal(err)
	}

	url := computeURL + "/v2.1/" + tenant.ID + "/servers"

	var server payloads.ComputeCreateServer
	server.Server.MaxInstances = 1
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
		t.Fatal(err)
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

	if servers.TotalServers != 1 {
		t.Fatal(err)
	}

	return
}
