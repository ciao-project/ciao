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

package main

import (
	"testing"
)

func TestCNCILaunch(t *testing.T) {
	testClient, client, instances := testStartWorkloadLaunchCNCI(t, 1)
	defer testClient.Shutdown()
	defer client.Shutdown()

	id := instances[0].TenantID

	// get CNCI info for this tenant
	cncis, err := ctl.ds.GetTenantCNCISummary("")
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range cncis {
		if c.TenantID != id {
			continue
		}

		if c.IPAddress == "" {
			t.Fatal("CNCI Info not updated")
		}
		return
	}

	t.Fatal("CNCI not found")
}
