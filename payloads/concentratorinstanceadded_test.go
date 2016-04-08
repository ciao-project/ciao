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

package payloads

import (
	"testing"

	"gopkg.in/yaml.v2"
)

const cnciUUID = "3390740c-dce9-48d6-b83a-a717417072ce"
const tenantUUID = "2491851d-dce9-48d6-b83a-a717417072ce"
const cnciIP = "10.1.2.3"
const cnciMAC = "CA:FE:C0:00:01:02"

const cnciAddedYaml = "" +
	"concentrator_instance_added:\n" +
	"  instance_uuid: " + cnciUUID + "\n" +
	"  tenant_uuid: " + tenantUUID + "\n" +
	"  concentrator_ip: " + cnciIP + "\n" +
	"  concentrator_mac: " + cnciMAC + "\n"

func TestConcentratorAddedUnmarshal(t *testing.T) {
	var cnciAdded EventConcentratorInstanceAdded

	err := yaml.Unmarshal([]byte(cnciAddedYaml), &cnciAdded)
	if err != nil {
		t.Error(err)
	}

	if cnciAdded.CNCIAdded.InstanceUUID != cnciUUID {
		t.Errorf("Wrong instance UUID field [%s]", cnciAdded.CNCIAdded.InstanceUUID)
	}

	if cnciAdded.CNCIAdded.TenantUUID != tenantUUID {
		t.Errorf("Wrong tenant UUID field [%s]", cnciAdded.CNCIAdded.TenantUUID)
	}

	if cnciAdded.CNCIAdded.ConcentratorIP != cnciIP {
		t.Errorf("Wrong CNCI IP field [%s]", cnciAdded.CNCIAdded.ConcentratorIP)
	}

	if cnciAdded.CNCIAdded.ConcentratorMAC != cnciMAC {
		t.Errorf("Wrong CNCI MAC field [%s]", cnciAdded.CNCIAdded.ConcentratorMAC)
	}
}

func TestConcentratorAddedMarshal(t *testing.T) {
	var cnciAdded EventConcentratorInstanceAdded

	cnciAdded.CNCIAdded.InstanceUUID = cnciUUID
	cnciAdded.CNCIAdded.TenantUUID = tenantUUID
	cnciAdded.CNCIAdded.ConcentratorIP = cnciIP
	cnciAdded.CNCIAdded.ConcentratorMAC = cnciMAC

	y, err := yaml.Marshal(&cnciAdded)
	if err != nil {
		t.Error(err)
	}

	if string(y) != cnciAddedYaml {
		t.Errorf("ConcentratorInstanceAdded marshalling failed\n[%s]\n vs\n[%s]", string(y), cnciAddedYaml)
	}
}
