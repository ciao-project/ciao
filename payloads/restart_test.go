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
	"fmt"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestRestartUnmarshal(t *testing.T) {
	restartYaml := `restart:
  instance_uuid: 0e8516d7-af2f-454a-87ed-072aeb9faf53
  image_uuid: 5beea770-1ef5-4c26-8a6c-2026fbc98e37
  workload_agent_uuid: d37e8dd5-3625-42bb-97b5-05291013abad
  fw_type: efi
  persistence: host
  requested_resources:
    - type: vcpus
      value: 2
      mandatory: true
    - type: mem_mb
      value: 1014
      mandatory: true
    - type: disk_mb
      value: 10000
      mandatory: true
  estimated_resources:
    - type: vcpus
      value: 1
    - type: mem_mb
      value: 128
    - type: disk_mb
      value: 4096
`
	var cmd Restart
	err := yaml.Unmarshal([]byte(restartYaml), &cmd)
	if err != nil {
		t.Error(err)
	}
}

func TestRestartMarshal(t *testing.T) {
	reqVcpus := RequestedResource{
		Type:      "vcpus",
		Value:     2,
		Mandatory: true,
	}
	reqMem := RequestedResource{
		Type:      "mem_mb",
		Value:     4096,
		Mandatory: true,
	}
	reqDisk := RequestedResource{
		Type:      "disk_mb",
		Value:     10000,
		Mandatory: true,
	}
	estVcpus := EstimatedResource{
		Type:  "vcpus",
		Value: 1,
	}
	estMem := EstimatedResource{
		Type:  "mem_mb",
		Value: 128,
	}
	estDisk := EstimatedResource{
		Type:  "disk_mb",
		Value: 4096,
	}
	var cmd Restart
	cmd.Restart.InstanceUUID = "3ad186a6-7343-4541-a747-78f0dddd9e3e"
	cmd.Restart.ImageUUID = "11a94b09-85b6-4434-9f4a-c19d863465f1"
	cmd.Restart.WorkloadAgentUUID = "d3acac98-17db-42dc-9fc3-6f737b7b73c2"
	cmd.Restart.RequestedResources = append(cmd.Restart.RequestedResources, reqVcpus)
	cmd.Restart.RequestedResources = append(cmd.Restart.RequestedResources, reqMem)
	cmd.Restart.RequestedResources = append(cmd.Restart.RequestedResources, reqDisk)
	cmd.Restart.EstimatedResources = append(cmd.Restart.EstimatedResources, estVcpus)
	cmd.Restart.EstimatedResources = append(cmd.Restart.EstimatedResources, estMem)
	cmd.Restart.EstimatedResources = append(cmd.Restart.EstimatedResources, estDisk)
	cmd.Restart.FWType = EFI
	cmd.Restart.InstancePersistence = Host

	y, err := yaml.Marshal(&cmd)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(y))
}

// make sure the yaml can be unmarshaled into the Restart struct with
// optional data not present
func TestRestartUnmarshalPartial(t *testing.T) {
	restartYaml := `restart:
  instance_uuid: a2675987-fa30-45ce-84a2-93ce67106f47
  workload_agent_uuid: 1ab3a664-d344-4a41-acf9-c94d8606e069
  fw_type: efi
  persistence: host
  requested_resources:
    - type: vcpus
      value: 2
      mandatory: true
`
	var cmd Restart
	err := yaml.Unmarshal([]byte(restartYaml), &cmd)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(cmd)

	var expectedCmd Restart
	expectedCmd.Restart.InstanceUUID = "a2675987-fa30-45ce-84a2-93ce67106f47"
	expectedCmd.Restart.WorkloadAgentUUID = "1ab3a664-d344-4a41-acf9-c94d8606e069"
	expectedCmd.Restart.FWType = EFI
	expectedCmd.Restart.InstancePersistence = Host
	vcpus := RequestedResource{
		Type:      "vcpus",
		Value:     2,
		Mandatory: true,
	}
	expectedCmd.Restart.RequestedResources = append(expectedCmd.Restart.RequestedResources, vcpus)

	if cmd.Restart.InstanceUUID != expectedCmd.Restart.InstanceUUID ||
		cmd.Restart.WorkloadAgentUUID != expectedCmd.Restart.WorkloadAgentUUID ||
		cmd.Restart.FWType != expectedCmd.Restart.FWType ||
		cmd.Restart.InstancePersistence != expectedCmd.Restart.InstancePersistence ||
		len(cmd.Restart.RequestedResources) != 1 ||
		len(expectedCmd.Restart.RequestedResources) != 1 ||
		cmd.Restart.RequestedResources[0].Type != expectedCmd.Restart.RequestedResources[0].Type ||
		cmd.Restart.RequestedResources[0].Value != expectedCmd.Restart.RequestedResources[0].Value ||
		cmd.Restart.RequestedResources[0].Mandatory != expectedCmd.Restart.RequestedResources[0].Mandatory {
		t.Error("Unexpected values in Restart")
	}

	fmt.Println(cmd)
}
