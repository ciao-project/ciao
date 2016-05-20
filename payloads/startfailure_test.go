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

func TestStartFailureUnmarshal(t *testing.T) {
	startFailureYaml := `instance_uuid: 2400bce6-ccc8-4a45-b2aa-b5cc3790077b
reason: full_cloud
`
	var error ErrorStartFailure
	err := yaml.Unmarshal([]byte(startFailureYaml), &error)
	if err != nil {
		t.Error(err)
	}

	if error.InstanceUUID != "2400bce6-ccc8-4a45-b2aa-b5cc3790077b" {
		t.Error("Wrong UUID field")
	}

	if error.Reason != FullCloud {
		t.Error("Wrong Error field")
	}
}

func TestStartFailureMarshal(t *testing.T) {
	error := ErrorStartFailure{
		InstanceUUID: uuid.Generate().String(),
		Reason:       FullCloud,
	}

	y, err := yaml.Marshal(&error)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(y))
}

func TestStartFailureString(t *testing.T) {
	var stringTests = []struct {
		r        StartFailureReason
		expected string
	}{
		{FullCloud, "Cloud is full"},
		{FullComputeNode, "Compute node is full"},
		{NoComputeNodes, "No compute node available"},
		{NoNetworkNodes, "No network node available"},
		{InvalidPayload, "YAML payload is corrupt"},
		{InvalidData, "Command section of YAML payload is corrupt or missing required information"},
		{AlreadyRunning, "Instance is already running"},
		{InstanceExists, "Instance already exists"},
		{ImageFailure, "Failed to create instance image"},
		{LaunchFailure, "Failed to launch instance"},
		{NetworkFailure, "Failed to create VNIC for instance"},
	}
	error := ErrorStartFailure{
		InstanceUUID: uuid.Generate().String(),
	}
	for _, test := range stringTests {
		error.Reason = test.r
		s := error.Reason.String()
		if s != test.expected {
			t.Errorf("expected \"%s\", got \"%s\"", test.expected, s)
		}
	}
}
