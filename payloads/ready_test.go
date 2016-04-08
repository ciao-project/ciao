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
	"github.com/docker/distribution/uuid"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestReadyUnmarshal(t *testing.T) {
	readyYaml := `node_uuid: 2400bce6-ccc8-4a45-b2aa-b5cc3790077b
mem_total_mb: 3896
mem_available_mb: 3896
disk_total_mb: 500000
disk_available_mb: 256000
load: 0
cpus_online: 4
`
	var cmd Ready
	err := yaml.Unmarshal([]byte(readyYaml), &cmd)
	if err != nil {
		t.Error(err)
	}
}

func TestReadyMarshal(t *testing.T) {
	cmd := Ready{
		NodeUUID:        uuid.Generate().String(),
		MemTotalMB:      3896,
		MemAvailableMB:  3896,
		DiskTotalMB:     500000,
		DiskAvailableMB: 256000,
		Load:            0,
		CpusOnline:      4,
	}

	y, err := yaml.Marshal(&cmd)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(y))
}

// make sure the yaml can be unmarshaled into the Ready struct
// when only some node stats are present
func TestReadyNodeNotAllStats(t *testing.T) {
	readyYaml := `node_uuid: 2400bce6-ccc8-4a45-b2aa-b5cc3790077b
load: 1
`
	var cmd Ready
	cmd.Init()

	err := yaml.Unmarshal([]byte(readyYaml), &cmd)
	if err != nil {
		t.Error(err)
	}

	expectedCmd := Ready{
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
		cmd.CpusOnline != expectedCmd.CpusOnline {
		t.Error("Unexpected values in Ready")
	}

	fmt.Println(cmd)
}
