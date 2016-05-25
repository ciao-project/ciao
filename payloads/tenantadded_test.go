/*
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
*/

package payloads_test

import (
	"testing"

	. "github.com/01org/ciao/payloads"
	"gopkg.in/yaml.v2"
)

const agentIP = "10.2.3.4"
const tenantSubnet = "10.2.0.0/16"
const subnetKey = "8"

const tenantAddedYaml = "" +
	"tenant_added:\n" +
	"  agent_uuid: " + agentUUID + "\n" +
	"  agent_ip: " + agentIP + "\n" +
	"  tenant_uuid: " + tenantUUID + "\n" +
	"  tenant_subnet: " + tenantSubnet + "\n" +
	"  concentrator_uuid: " + cnciUUID + "\n" +
	"  concentrator_ip: " + cnciIP + "\n" +
	"  subnet_key: " + subnetKey + "\n"

const tenantRemovedYaml = "" +
	"tenant_removed:\n" +
	"  agent_uuid: " + agentUUID + "\n" +
	"  agent_ip: " + agentIP + "\n" +
	"  tenant_uuid: " + tenantUUID + "\n" +
	"  tenant_subnet: " + tenantSubnet + "\n" +
	"  concentrator_uuid: " + cnciUUID + "\n" +
	"  concentrator_ip: " + cnciIP + "\n" +
	"  subnet_key: " + subnetKey + "\n"

func TestTenantAddedUnmarshal(t *testing.T) {
	var tenantAdded EventTenantAdded

	err := yaml.Unmarshal([]byte(tenantAddedYaml), &tenantAdded)
	if err != nil {
		t.Error(err)
	}

	if tenantAdded.TenantAdded.AgentUUID != agentUUID {
		t.Errorf("Wrong agent UUID field [%s]", tenantAdded.TenantAdded.AgentUUID)
	}

	if tenantAdded.TenantAdded.AgentIP != agentIP {
		t.Errorf("Wrong agent IP field [%s]", tenantAdded.TenantAdded.AgentIP)
	}

	if tenantAdded.TenantAdded.TenantUUID != tenantUUID {
		t.Errorf("Wrong tenant UUID field [%s]", tenantAdded.TenantAdded.TenantUUID)
	}

	if tenantAdded.TenantAdded.TenantSubnet != tenantSubnet {
		t.Errorf("Wrong tenant subnet field [%s]", tenantAdded.TenantAdded.TenantSubnet)
	}

	if tenantAdded.TenantAdded.ConcentratorUUID != cnciUUID {
		t.Errorf("Wrong CNCI UUID field [%s]", tenantAdded.TenantAdded.ConcentratorUUID)
	}

	if tenantAdded.TenantAdded.ConcentratorIP != cnciIP {
		t.Errorf("Wrong CNCI IP field [%s]", tenantAdded.TenantAdded.ConcentratorIP)
	}

}

func TestTenantRemovedUnmarshal(t *testing.T) {
	var tenantRemoved EventTenantRemoved

	err := yaml.Unmarshal([]byte(tenantRemovedYaml), &tenantRemoved)
	if err != nil {
		t.Error(err)
	}

	if tenantRemoved.TenantRemoved.AgentUUID != agentUUID {
		t.Errorf("Wrong agent UUID field [%s]", tenantRemoved.TenantRemoved.AgentUUID)
	}

	if tenantRemoved.TenantRemoved.AgentIP != agentIP {
		t.Errorf("Wrong agent IP field [%s]", tenantRemoved.TenantRemoved.AgentIP)
	}

	if tenantRemoved.TenantRemoved.TenantUUID != tenantUUID {
		t.Errorf("Wrong tenant UUID field [%s]", tenantRemoved.TenantRemoved.TenantUUID)
	}

	if tenantRemoved.TenantRemoved.TenantSubnet != tenantSubnet {
		t.Errorf("Wrong tenant subnet field [%s]", tenantRemoved.TenantRemoved.TenantSubnet)
	}

	if tenantRemoved.TenantRemoved.ConcentratorUUID != cnciUUID {
		t.Errorf("Wrong CNCI UUID field [%s]", tenantRemoved.TenantRemoved.ConcentratorUUID)
	}

	if tenantRemoved.TenantRemoved.ConcentratorIP != cnciIP {
		t.Errorf("Wrong CNCI IP field [%s]", tenantRemoved.TenantRemoved.ConcentratorIP)
	}

}

func TestTenantAddedMarshal(t *testing.T) {
	var tenantAdded EventTenantAdded

	tenantAdded.TenantAdded.AgentUUID = agentUUID
	tenantAdded.TenantAdded.AgentIP = agentIP
	tenantAdded.TenantAdded.TenantUUID = tenantUUID
	tenantAdded.TenantAdded.TenantSubnet = tenantSubnet
	tenantAdded.TenantAdded.ConcentratorUUID = cnciUUID
	tenantAdded.TenantAdded.ConcentratorIP = cnciIP
	tenantAdded.TenantAdded.SubnetKey = 8

	y, err := yaml.Marshal(&tenantAdded)
	if err != nil {
		t.Error(err)
	}

	if string(y) != tenantAddedYaml {
		t.Errorf("TenantAdded marshalling failed\n[%s]\n vs\n[%s]", string(y), tenantAddedYaml)
	}
}

func TestTenantRemovedMarshal(t *testing.T) {
	var tenantRemoved EventTenantRemoved

	tenantRemoved.TenantRemoved.AgentUUID = agentUUID
	tenantRemoved.TenantRemoved.AgentIP = agentIP
	tenantRemoved.TenantRemoved.TenantUUID = tenantUUID
	tenantRemoved.TenantRemoved.TenantSubnet = tenantSubnet
	tenantRemoved.TenantRemoved.ConcentratorUUID = cnciUUID
	tenantRemoved.TenantRemoved.ConcentratorIP = cnciIP
	tenantRemoved.TenantRemoved.SubnetKey = 8

	y, err := yaml.Marshal(&tenantRemoved)
	if err != nil {
		t.Error(err)
	}

	if string(y) != tenantRemovedYaml {
		t.Errorf("TenantRemoved marshalling failed\n[%s]\n vs\n[%s]", string(y), tenantRemovedYaml)
	}
}
