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

const cnciInstanceData = "scheduler_addr: 192.168.42.5\n"

func TestCNCIInstanceUnmarshal(t *testing.T) {
	var config CNCIInstanceConfig
	err := yaml.Unmarshal([]byte(cnciInstanceData), &config)
	if err != nil {
		t.Error(err)
	}

	if config.SchedulerAddr != "192.168.42.5" {
		t.Errorf("Wrong ADDR field [%s]", config.SchedulerAddr)
	}
}

func TestCNCIInstanceMarshal(t *testing.T) {
	config := CNCIInstanceConfig{SchedulerAddr: "192.168.42.5"}

	y, err := yaml.Marshal(&config)
	if err != nil {
		t.Error(err)
	}

	if string(y) != cnciInstanceData {
		t.Errorf("CNCIInstance marshalling failed\n[%s]\n vs\n[%s]", string(y), cnciInstanceData)
	}
}
