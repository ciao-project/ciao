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
	"gopkg.in/yaml.v2"
)

const insDelUUID = "3390740c-dce9-48d6-b83a-a717417072ce"

const insDelYaml = "" +
	"instance_deleted:\n" +
	"  instance_uuid: " + insDelUUID + "\n"

func TestInstanceDeletedUnmarshal(t *testing.T) {
	var insDel EventInstanceDeleted
	err := yaml.Unmarshal([]byte(insDelYaml), &insDel)
	if err != nil {
		t.Error(err)
	}

	if insDel.InstanceDeleted.InstanceUUID != insDelUUID {
		t.Errorf("Wrong instance UUID field [%s]", insDel.InstanceDeleted.InstanceUUID)
	}
}

func TestInstanceDeletedMarshal(t *testing.T) {
	var insDel EventInstanceDeleted

	insDel.InstanceDeleted.InstanceUUID = instanceUUID

	y, err := yaml.Marshal(&insDel)
	if err != nil {
		t.Error(err)
	}

	if string(y) != insDelYaml {
		t.Errorf("InstanceDeleted marshalling failed\n[%s]\n vs\n[%s]", string(y), insDelYaml)
	}
}
