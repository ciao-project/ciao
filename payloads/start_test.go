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

	. "github.com/ciao-project/ciao/payloads"
	"github.com/ciao-project/ciao/testutil"
	"gopkg.in/yaml.v2"
)

func TestStartUnmarshal(t *testing.T) {
	var cmd Start
	err := yaml.Unmarshal([]byte(testutil.StartYaml), &cmd)
	if err != nil {
		t.Error(err)
	}
}

func TestStartMarshal(t *testing.T) {
	var cmd Start
	cmd.Start.TenantUUID = testutil.TenantUUID
	cmd.Start.InstanceUUID = testutil.InstanceUUID
	cmd.Start.DockerImage = testutil.DockerImage
	cmd.Start.Requirements.VCPUs = 2
	cmd.Start.Requirements.MemMB = 4096
	cmd.Start.FWType = EFI
	cmd.Start.InstancePersistence = Host
	cmd.Start.VMType = QEMU

	y, err := yaml.Marshal(&cmd)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.StartYaml {
		t.Errorf("Start marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.StartYaml)
	}
}

// make sure the yaml can be unmarshaled into the Start struct with
// optional data not present
func TestStartUnmarshalPartial(t *testing.T) {
	var cmd Start
	err := yaml.Unmarshal([]byte(testutil.PartialStartYaml), &cmd)
	if err != nil {
		t.Error(err)
	}

	var expectedCmd Start
	expectedCmd.Start.InstanceUUID = testutil.InstanceUUID
	expectedCmd.Start.DockerImage = testutil.DockerImage
	expectedCmd.Start.FWType = EFI
	expectedCmd.Start.InstancePersistence = Host
	expectedCmd.Start.VMType = QEMU
	expectedCmd.Start.Requirements.VCPUs = 2

	if cmd.Start.InstanceUUID != expectedCmd.Start.InstanceUUID ||
		cmd.Start.DockerImage != expectedCmd.Start.DockerImage ||
		cmd.Start.FWType != expectedCmd.Start.FWType ||
		cmd.Start.InstancePersistence != expectedCmd.Start.InstancePersistence ||
		cmd.Start.VMType != expectedCmd.Start.VMType ||
		cmd.Start.Requirements.VCPUs != expectedCmd.Start.Requirements.VCPUs {
		t.Error("Unexpected values in Start")
	}
}
