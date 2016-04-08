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
	"gopkg.in/yaml.v2"
	"testing"
)

const evacAgentUUID = "64803ffa-fb47-49fa-8191-15d2c34e4dd3"
const evacYaml = "" +
	"evacuate:\n" +
	"  workload_agent_uuid: " + evacAgentUUID + "\n"

func TestEvacMarshal(t *testing.T) {
	var cmd Evacuate
	cmd.Evacuate.WorkloadAgentUUID = evacAgentUUID

	y, err := yaml.Marshal(&cmd)
	if err != nil {
		t.Error(err)
	}

	if string(y) != evacYaml {
		t.Errorf("EVACUATE marshalling failed\n[%s]\n vs\n[%s]", string(y), evacYaml)
	}
}

func TestEvacUnmarshal(t *testing.T) {
	var cmd Evacuate
	err := yaml.Unmarshal([]byte(evacYaml), &cmd)
	if err != nil {
		t.Error(err)
	}

	if cmd.Evacuate.WorkloadAgentUUID != evacAgentUUID {
		t.Errorf("Wrong Agent UUID field [%s]", cmd.Evacuate.WorkloadAgentUUID)
	}
}
