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

	"github.com/ciao-project/ciao/ssntp"
)

func TestCNCIInitializeCtrls(t *testing.T) {
	err := initializeCNCICtrls(ctl)
	if err != nil {
		t.Fatal(err)
	}

	ts, err := ctl.ds.GetAllTenants()
	if err != nil {
		t.Fatal(err)
	}

	for _, tenant := range ts {
		if tenant.CNCIctrl == nil {
			t.Fatal("CNCIctrl not initialized properly")
		}
	}
}

func TestCNCILaunch(t *testing.T) {
	testClient, client, instances := testStartWorkloadLaunchCNCI(t, 1)
	defer testClient.Shutdown()
	defer client.Shutdown()

	id := instances[0].TenantID

	// get CNCI info for this tenant
	instances, err := ctl.ds.GetTenantCNCIs(id)
	if err != nil {
		t.Fatal(err)
	}

	if len(instances) != 1 {
		t.Fatal("Incorrect number of CNCIs")
	}

	if instances[0].IPAddress == "" {
		t.Fatal("CNCI Info not updated")
	}

	tenant, err := ctl.ds.GetTenant(id)
	if err != nil {
		t.Fatal(err)
	}

	if !tenant.CNCIctrl.Active(instances[0].ID) {
		t.Fatal(err)
	}
}

func TestCNCIRemoved(t *testing.T) {
	netClient, client, instances := testStartWorkloadLaunchCNCI(t, 1)
	defer client.Shutdown()
	defer netClient.Shutdown()

	sendStatsCmd(client, t)
	sendStatsCmd(netClient, t)

	instanceID := instances[0].ID
	tenantID := instances[0].TenantID

	// get the tenant
	tenant, err := ctl.ds.GetTenant(tenantID)
	if err != nil {
		t.Fatal(err)
	}

	// get CNCI for this instance.
	cnci, err := tenant.CNCIctrl.GetInstanceCNCI(instanceID)
	if err != nil {
		t.Fatal(err)
	}

	// confirm that instance is added.
	_, err = ctl.ds.GetInstance(instanceID)
	if err != nil {
		t.Fatal("Instance not actually created")
	}

	serverCh := server.AddCmdChan(ssntp.DELETE)
	clientCh := client.AddCmdChan(ssntp.DELETE)
	netClientCh := netClient.AddCmdChan(ssntp.DELETE)

	err = ctl.deleteInstance(instanceID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = server.GetCmdChanResult(serverCh, ssntp.DELETE)
	if err != nil {
		t.Fatal(err)
	}

	_, err = server.GetCmdChanResult(clientCh, ssntp.DELETE)
	if err != nil {
		t.Fatal(err)
	}

	controllerCh := wrappedClient.addEventChan(ssntp.InstanceDeleted)
	go client.SendDeleteEvent(instances[0].ID)

	err = wrappedClient.getEventChan(controllerCh, ssntp.InstanceDeleted)
	if err != nil {
		t.Fatal(err)
	}

	sendStatsCmd(client, t)

	_, err = ctl.ds.GetInstance(instanceID)
	if err == nil {
		t.Fatal("instance was not deleted")
	}

	tenant, err = ctl.ds.GetTenant(tenantID)
	if err != nil {
		t.Fatal(err)
	}

	CNCIID := cnci.ID

	// call remove subnet directly to remove the cnci.
	go func() {
		err = tenant.CNCIctrl.RemoveSubnet(4096)
		if err != nil {
			t.Fatal(err)
		}
	}()

	_, err = server.GetCmdChanResult(serverCh, ssntp.DELETE)
	if err != nil {
		t.Fatal(err)
	}

	_, err = server.GetCmdChanResult(netClientCh, ssntp.DELETE)
	if err != nil {
		t.Fatal(err)
	}

	go netClient.SendDeleteEvent(CNCIID)

	err = wrappedClient.getEventChan(controllerCh, ssntp.InstanceDeleted)
	if err != nil {
		t.Fatal(err)
	}
}
