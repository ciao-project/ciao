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
