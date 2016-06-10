/* // Copyright (c) 2016 Intel Corporation
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
	"github.com/01org/ciao/testutil"
	"gopkg.in/yaml.v2"
)

func TestPublicIPAssignedUnmarshal(t *testing.T) {
	var assignedIP EventPublicIPAssigned

	err := yaml.Unmarshal([]byte(testutil.AssignedIPYaml), &assignedIP)
	if err != nil {
		t.Error(err)
	}

	if assignedIP.AssignedIP.ConcentratorUUID != testutil.CNCIUUID {
		t.Errorf("Wrong concentrator UUID field [%s]", assignedIP.AssignedIP.ConcentratorUUID)
	}

	if assignedIP.AssignedIP.InstanceUUID != testutil.InstanceUUID {
		t.Errorf("Wrong instance UUID field [%s]", assignedIP.AssignedIP.InstanceUUID)
	}

	if assignedIP.AssignedIP.PublicIP != testutil.InstancePublicIP {
		t.Errorf("Wrong public IP field [%s]", assignedIP.AssignedIP.PublicIP)
	}

	if assignedIP.AssignedIP.PrivateIP != testutil.InstancePrivateIP {
		t.Errorf("Wrong private IP field [%s]", assignedIP.AssignedIP.PrivateIP)
	}
}

func TestPublicIPAssignedMarshal(t *testing.T) {
	var assignedIP EventPublicIPAssigned

	assignedIP.AssignedIP.ConcentratorUUID = testutil.CNCIUUID
	assignedIP.AssignedIP.InstanceUUID = testutil.InstanceUUID
	assignedIP.AssignedIP.PublicIP = testutil.InstancePublicIP
	assignedIP.AssignedIP.PrivateIP = testutil.InstancePrivateIP

	y, err := yaml.Marshal(&assignedIP)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.AssignedIPYaml {
		t.Errorf("PublicIPAssigned marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.AssignedIPYaml)
	}
}
