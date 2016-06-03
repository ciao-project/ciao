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
	"fmt"
	"testing"

	. "github.com/01org/ciao/payloads"
	"github.com/docker/distribution/uuid"
	"gopkg.in/yaml.v2"
)

func TestStatsUnmarshal(t *testing.T) {
	statsYaml := `node_uuid: 2400bce6-ccc8-4a45-b2aa-b5cc3790077b
status: READY
mem_total_mb: 3896
mem_available_mb: 3896
disk_total_mb: 500000
disk_available_mb: 256000
load: 0
cpus_online: 4
hostname: test
networks:
  - ip: 192.168.1.1
    mac: 02:00:15:03:6f:49
  - ip: 10.168.1.1
    mac: 02:00:8c:ba:f9:45
instances:
  - instance_uuid: fe2970fa-7b36-460b-8b79-9eb4745e62f2
    state: running
    memory_usage_mb: 40
    disk_usage_mb: 2
    cpu_usage: 90
    ssh_ip: ""
    ssh_port: 0
  - instance_uuid: cbda5bd8-33bd-4d39-9f52-ace8c9f0b99c
    state: running
    memory_usage_mb: 50
    disk_usage_mb: 10
    cpu_usage: 0
    ssh_ip: 172.168.2.2
    ssh_port: 8768
  - instance_uuid: 1f5b2fe6-4493-4561-904a-8f4e956218d9
    state: exited
    memory_usage_mb: -1
    disk_usage_mb: 2
    cpu_usage: -1
`
	var cmd Stat
	err := yaml.Unmarshal([]byte(statsYaml), &cmd)
	if err != nil {
		t.Error(err)
	}
}

func TestStatsMarshal(t *testing.T) {
	nstats := NetworkStat{
		NodeIP:  "192.168.1.1",
		NodeMAC: "02:00:0f:57:39:45",
	}
	istats := InstanceStat{
		InstanceUUID:  uuid.Generate().String(),
		State:         Running,
		MemoryUsageMB: 40,
		DiskUsageMB:   20,
		CPUUsage:      70,
		SSHIP:         "172.168.0.4",
		SSHPort:       33004,
	}
	cmd := Stat{
		NodeUUID:        uuid.Generate().String(),
		Status:          "READY",
		MemTotalMB:      3896,
		MemAvailableMB:  3896,
		DiskTotalMB:     500000,
		DiskAvailableMB: 256000,
		Load:            0,
		CpusOnline:      4,
		NodeHostName:    "test",
	}
	cmd.Instances = append(cmd.Instances, istats)
	cmd.Networks = append(cmd.Networks, nstats)

	y, err := yaml.Marshal(&cmd)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(y))
}

// make sure the yaml can be unmarshaled into the Stat struct with
// no instances present
func TestStatsNodeOnly(t *testing.T) {
	statsYaml := `node_uuid: 2400bce6-ccc8-4a45-b2aa-b5cc3790077b
mem_total_mb: 3896
mem_available_mb: 3896
disk_total_mb: 500000
disk_available_mb: 256000
load: 0
cpus_online: 4
hostname: test
networks:
  - ip: 192.168.1.1
    mac: 02:00:15:03:6f:49
`
	var cmd Stat
	err := yaml.Unmarshal([]byte(statsYaml), &cmd)
	if err != nil {
		t.Error(err)
	}

	expectedCmd := Stat{
		NodeUUID:        "2400bce6-ccc8-4a45-b2aa-b5cc3790077b",
		MemTotalMB:      3896,
		MemAvailableMB:  3896,
		DiskTotalMB:     500000,
		DiskAvailableMB: 256000,
		Load:            0,
		CpusOnline:      4,
		NodeHostName:    "test",
		Networks: []NetworkStat{
			{
				NodeIP:  "192.168.1.1",
				NodeMAC: "02:00:15:03:6f:49",
			},
		},
	}
	if cmd.NodeUUID != expectedCmd.NodeUUID ||
		cmd.MemTotalMB != expectedCmd.MemTotalMB ||
		cmd.MemAvailableMB != expectedCmd.MemAvailableMB ||
		cmd.DiskTotalMB != expectedCmd.DiskTotalMB ||
		cmd.DiskAvailableMB != expectedCmd.DiskAvailableMB ||
		cmd.Load != expectedCmd.Load ||
		cmd.CpusOnline != expectedCmd.CpusOnline ||
		cmd.NodeHostName != expectedCmd.NodeHostName ||
		len(cmd.Networks) != 1 ||
		cmd.Networks[0] != expectedCmd.Networks[0] ||
		cmd.Instances != nil {
		t.Error("Unexpected values in Stat")
	}

	fmt.Println(cmd)
}

// make sure the yaml can be unmarshaled into the Stat struct
// when only some node stats are present
func TestStatsNodeNotAllStats(t *testing.T) {
	statsYaml := `node_uuid: 2400bce6-ccc8-4a45-b2aa-b5cc3790077b
load: 1
`
	var cmd Stat
	cmd.Init()

	err := yaml.Unmarshal([]byte(statsYaml), &cmd)
	if err != nil {
		t.Error(err)
	}

	expectedCmd := Stat{
		NodeUUID:        "2400bce6-ccc8-4a45-b2aa-b5cc3790077b",
		MemTotalMB:      -1,
		MemAvailableMB:  -1,
		DiskTotalMB:     -1,
		DiskAvailableMB: -1,
		Load:            1,
		CpusOnline:      -1,
	}
	if cmd.NodeUUID != expectedCmd.NodeUUID ||
		cmd.MemTotalMB != expectedCmd.MemTotalMB ||
		cmd.MemAvailableMB != expectedCmd.MemAvailableMB ||
		cmd.DiskTotalMB != expectedCmd.DiskTotalMB ||
		cmd.DiskAvailableMB != expectedCmd.DiskAvailableMB ||
		cmd.Load != expectedCmd.Load ||
		cmd.CpusOnline != expectedCmd.CpusOnline ||
		cmd.NodeHostName != expectedCmd.NodeHostName ||
		cmd.Networks != nil ||
		cmd.Instances != nil {
		t.Error("Unexpected values in Stat")
	}

	fmt.Println(cmd)
}
