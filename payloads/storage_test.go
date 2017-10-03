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

func TestAttachVolumeUnmarshal(t *testing.T) {
	var attach AttachVolume
	err := yaml.Unmarshal([]byte(testutil.AttachVolumeYaml), &attach)
	if err != nil {
		t.Error(err)
	}

	if attach.Attach.InstanceUUID != testutil.InstanceUUID {
		t.Errorf("Wrong instance UUID field [%s]", attach.Attach.InstanceUUID)
	}

	if attach.Attach.VolumeUUID != testutil.VolumeUUID {
		t.Errorf("Wrong Volume UUID field [%s]", attach.Attach.VolumeUUID)
	}

	if attach.Attach.WorkloadAgentUUID != testutil.AgentUUID {
		t.Errorf("Wrong WorkloadAgentUUID field [%s]", attach.Attach.WorkloadAgentUUID)
	}
}

func TestAttachVolmeMarshal(t *testing.T) {
	var attach AttachVolume
	attach.Attach.InstanceUUID = testutil.InstanceUUID
	attach.Attach.VolumeUUID = testutil.VolumeUUID
	attach.Attach.WorkloadAgentUUID = testutil.AgentUUID

	y, err := yaml.Marshal(&attach)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.AttachVolumeYaml {
		t.Errorf("AttachVolume marshalling failed\n[%s]\n vs\n[%s]",
			string(y), testutil.AttachVolumeYaml)
	}
}
