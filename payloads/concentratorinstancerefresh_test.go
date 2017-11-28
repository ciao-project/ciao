/*
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
*/

package payloads_test

import (
	"testing"

	. "github.com/ciao-project/ciao/payloads"
	"github.com/ciao-project/ciao/testutil"
	"gopkg.in/yaml.v2"
)

func TestConcentratorRefreshUnmarshal(t *testing.T) {
	var cnciRefresh CommandCNCIRefresh

	err := yaml.Unmarshal([]byte(testutil.CNCIRefreshYaml), &cnciRefresh)
	if err != nil {
		t.Error(err)
	}

	if cnciRefresh.Command.CNCIUUID != testutil.CNCIUUID {
		t.Errorf("Incorrect CNCI UUID [%s]", cnciRefresh.Command.CNCIUUID)
	}

	if len(cnciRefresh.Command.CNCIList) != 1 {
		t.Errorf("Incorrect length of CNCI list [%d]", len(cnciRefresh.Command.CNCIList))
	}

	newCNCI := cnciRefresh.Command.CNCIList[0]

	if newCNCI.PhysicalIP != "10.10.10.1" {
		t.Errorf("Wrong PhysicalIP field [%s]", newCNCI.PhysicalIP)
	}

	if newCNCI.Subnet != "172.16.0.0/24" {
		t.Errorf("Wrong subnet field [%s]", newCNCI.Subnet)
	}

	if newCNCI.TunnelIP != "192.168.0.0" {
		t.Errorf("Wrong Tunnel IP field [%s]", newCNCI.TunnelIP)
	}

	if newCNCI.TunnelID != testutil.CNCITunnelID {
		t.Errorf("Wrong Tunnel ID field [%d]", newCNCI.TunnelID)
	}
}

func TestConcentratorRefreshMarshal(t *testing.T) {
	var cnciRefresh CommandCNCIRefresh
	var newCNCI CNCINet

	newCNCI.PhysicalIP = "10.10.10.1"
	newCNCI.Subnet = "172.16.0.0/24"
	newCNCI.TunnelIP = "192.168.0.0"
	newCNCI.TunnelID = testutil.CNCITunnelID

	cnciRefresh.Command.CNCIList = append(cnciRefresh.Command.CNCIList, newCNCI)
	cnciRefresh.Command.CNCIUUID = testutil.CNCIUUID

	y, err := yaml.Marshal(&cnciRefresh)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.CNCIRefreshYaml {
		t.Errorf("ConcentratorInstanceRefresh marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.CNCIRefreshYaml)
	}
}
