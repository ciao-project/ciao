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

var configureYaml = "" +
	"configure:\n" +
	"  imageservice:\n" +
	"    type: " + Glance.String() + "\n" +
	"    url: " + glanceURL + "\n" +
	"  identityservice:\n" +
	"    type: " + Keystone.String() + "\n" +
	"    url: " + keystoneURL + "\n"

func TestConfigureUnmarshal(t *testing.T) {
	var cfg CommandConfigure

	err := yaml.Unmarshal([]byte(configureYaml), &cfg)
	if err != nil {
		t.Error(err)
	}

	if cfg.Configure.ImageService.Type != Glance {
		t.Errorf("Wrong image service type [%s]", cfg.Configure.ImageService.Type)
	}
}

func TestConfigureMarshal(t *testing.T) {
	var cfg CommandConfigure

	cfg.Configure.ImageService.Type = Glance
	cfg.Configure.ImageService.URL = glanceURL
	cfg.Configure.IdentityService.Type = Keystone
	cfg.Configure.IdentityService.URL = keystoneURL

	y, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Error(err)
	}

	if string(y) != configureYaml {
		t.Errorf("CONFIGURE marshalling failed\n[%s]\n vs\n[%s]", string(y), configureYaml)
	}
}
