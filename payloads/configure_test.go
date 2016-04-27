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
	"testing"

	"gopkg.in/yaml.v2"
)

const keystoneURL = "http://keystone.example.com"
const glanceURL = "http://glance.example.com"
const computeNet = "192.168.1.110"
const mgmtNet = "192.168.1.111"

var configureYaml = "" +
	"configure:\n" +
	"  launcher:\n" +
	"    compute_net: " + computeNet + "\n" +
	"    mgmt_net: " + mgmtNet + "\n" +
	"    disk_limit: false\n" +
	"    mem_limit: false\n" +
	"  image_service:\n" +
	"    type: " + Glance.String() + "\n" +
	"    url: " + glanceURL + "\n" +
	"  identity_service:\n" +
	"    type: " + Keystone.String() + "\n" +
	"    url: " + keystoneURL + "\n"

func TestConfigureUnmarshal(t *testing.T) {
	var cfg Configure

	err := yaml.Unmarshal([]byte(configureYaml), &cfg)
	if err != nil {
		t.Error(err)
	}

	if cfg.Configure.ImageService.Type != Glance {
		t.Errorf("Wrong image service type [%s]", cfg.Configure.ImageService.Type)
	}

	if cfg.Configure.IdentityService.Type != Keystone {
		t.Errorf("Wrong identity service type [%s]", cfg.Configure.IdentityService.Type)
	}

	if cfg.Configure.Launcher.ManagementNetwork != mgmtNet {
		t.Errorf("Wrong launcher management network [%s]", cfg.Configure.Launcher.ManagementNetwork)
	}

	if cfg.Configure.Launcher.ComputeNetwork != computeNet {
		t.Errorf("Wrong launcher compute network [%s]", cfg.Configure.Launcher.ComputeNetwork)
	}
}

func TestConfigureMarshal(t *testing.T) {
	var cfg Configure

	cfg.Configure.ImageService.Type = Glance
	cfg.Configure.ImageService.URL = glanceURL

	cfg.Configure.IdentityService.Type = Keystone
	cfg.Configure.IdentityService.URL = keystoneURL

	cfg.Configure.Launcher.ComputeNetwork = computeNet
	cfg.Configure.Launcher.ManagementNetwork = mgmtNet
	cfg.Configure.Launcher.DiskLimit = false
	cfg.Configure.Launcher.MemoryLimit = false

	y, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Error(err)
	}

	if string(y) != configureYaml {
		t.Errorf("CONFIGURE marshalling failed\n[%s]\n vs\n[%s]", string(y), configureYaml)
	}
}
