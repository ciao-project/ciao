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

const assignedIPYaml = "" +
	"public_ip_assigned:\n" +
	"  concentrator_uuid: " + cnciUUID + "\n" +
	"  instance_uuid: " + instanceUUID + "\n" +
	"  public_ip: " + instancePublicIP + "\n" +
	"  private_ip: " + instancePrivateIP + "\n"

func TestPublicIPAssignedUnmarshal(t *testing.T) {
	var assignedIP EventPublicIPAssigned

	err := yaml.Unmarshal([]byte(assignedIPYaml), &assignedIP)
	if err != nil {
		t.Error(err)
	}

	if assignedIP.AssignedIP.ConcentratorUUID != cnciUUID {
		t.Errorf("Wrong concentrator UUID field [%s]", assignedIP.AssignedIP.ConcentratorUUID)
	}

	if assignedIP.AssignedIP.InstanceUUID != instanceUUID {
		t.Errorf("Wrong instance UUID field [%s]", assignedIP.AssignedIP.InstanceUUID)
	}

	if assignedIP.AssignedIP.PublicIP != instancePublicIP {
		t.Errorf("Wrong public IP field [%s]", assignedIP.AssignedIP.PublicIP)
	}

	if assignedIP.AssignedIP.PrivateIP != instancePrivateIP {
		t.Errorf("Wrong private IP field [%s]", assignedIP.AssignedIP.PrivateIP)
	}
}

func TestPublicIPAssignedMarshal(t *testing.T) {
	var assignedIP EventPublicIPAssigned

	assignedIP.AssignedIP.ConcentratorUUID = cnciUUID
	assignedIP.AssignedIP.InstanceUUID = instanceUUID
	assignedIP.AssignedIP.PublicIP = instancePublicIP
	assignedIP.AssignedIP.PrivateIP = instancePrivateIP

	y, err := yaml.Marshal(&assignedIP)
	if err != nil {
		t.Error(err)
	}

	if string(y) != assignedIPYaml {
		t.Errorf("PublicIPAssigned marshalling failed\n[%s]\n vs\n[%s]", string(y), assignedIPYaml)
	}
}
