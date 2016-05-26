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
	"github.com/01org/ciao/testutil"
	"gopkg.in/yaml.v2"
)

const instanceUUID = "3390740c-dce9-48d6-b83a-a717417072ce"
const agentUUID = "59460b8a-5f53-4e3e-b5ce-b71fed8c7e64"

func TestStopUnmarshal(t *testing.T) {
	var stop Stop
	err := yaml.Unmarshal([]byte(testutil.StopYaml), &stop)
	if err != nil {
		t.Error(err)
	}

	if stop.Stop.InstanceUUID != instanceUUID {
		t.Errorf("Wrong instance UUID field [%s]", stop.Stop.InstanceUUID)
	}

	if stop.Stop.WorkloadAgentUUID != agentUUID {
		t.Errorf("Wrong Agent UUID field [%s]", stop.Stop.WorkloadAgentUUID)
	}
}

func TestDeleteUnmarshal(t *testing.T) {
	var delete Delete
	err := yaml.Unmarshal([]byte(testutil.DeleteYaml), &delete)
	if err != nil {
		t.Error(err)
	}

	if delete.Delete.InstanceUUID != instanceUUID {
		t.Errorf("Wrong instance UUID field [%s]", delete.Delete.InstanceUUID)
	}

	if delete.Delete.WorkloadAgentUUID != agentUUID {
		t.Errorf("Wrong Agent UUID field [%s]", delete.Delete.WorkloadAgentUUID)
	}
}

func TestStopMarshal(t *testing.T) {
	var stop Stop
	stop.Stop.InstanceUUID = instanceUUID
	stop.Stop.WorkloadAgentUUID = agentUUID

	y, err := yaml.Marshal(&stop)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.StopYaml {
		t.Errorf("STOP marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.StopYaml)
	}
}

func TestDeleteMarshal(t *testing.T) {
	var delete Delete
	delete.Delete.InstanceUUID = instanceUUID
	delete.Delete.WorkloadAgentUUID = agentUUID

	y, err := yaml.Marshal(&delete)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.DeleteYaml {
		t.Errorf("DELETE marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.DeleteYaml)
	}
}
