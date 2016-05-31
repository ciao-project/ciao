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

	"fmt"
	. "github.com/01org/ciao/payloads"
	"gopkg.in/yaml.v2"
)

const keystoneURL = "http://keystone.example.com"
const glanceURL = "http://glance.example.com"
const computeNet = "192.168.1.110"
const mgmtNet = "192.168.1.111"
const storageURI = "/etc/ciao/ciao.json"
const identityUser = "controller"
const identityPassword = "ciao"
const computePort = 443
const httpsKey = "/etc/pki/ciao/compute_key.pem"
const httpsCACert = "/etc/pki/ciao/compute_ca.pem"

var configureYaml = "" +
	"configure:\n" +
	"  scheduler:\n" +
	"    storage_type: " + Filesystem.String() + "\n" +
	"    storage_uri: " + storageURI + "\n" +
	"  controller:\n" +
	"    compute_port: " + fmt.Sprintf("%d", computePort) + "\n" +
	"    compute_ca: " + httpsCACert + "\n" +
	"    compute_cert: " + httpsKey + "\n" +
	"    identity_user: " + identityUser + "\n" +
	"    identity_password: " + identityPassword + "\n" +
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

	if cfg.Configure.Scheduler.ConfigStorageType != Filesystem {
		t.Errorf("Wrong scheduler storage type [%s]", cfg.Configure.Scheduler.ConfigStorageType)
	}

	if cfg.Configure.Controller.ComputePort != computePort {
		t.Errorf("Wrong controller compute port [%d]", cfg.Configure.Controller.ComputePort)
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

	cfg.Configure.Controller.ComputePort = computePort
	cfg.Configure.Controller.HTTPSCACert = httpsCACert
	cfg.Configure.Controller.HTTPSKey = httpsKey
	cfg.Configure.Controller.IdentityUser = identityUser
	cfg.Configure.Controller.IdentityPassword = identityPassword

	cfg.Configure.Scheduler.ConfigStorageType = Filesystem
	cfg.Configure.Scheduler.ConfigStorageURI = storageURI

	y, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Error(err)
	}

	if string(y) != configureYaml {
		t.Errorf("CONFIGURE marshalling failed\n[%s]\n vs\n[%s]", string(y), configureYaml)
	}
}

func TestConfigureStorageTypeString(t *testing.T) {
	var stringTests = []struct {
		s        StorageType
		expected string
	}{
		{Filesystem, "file"},
		{Etcd, "etcd"},
	}
	for _, test := range stringTests {
		obj := test.s
		out := obj.String()
		if out != test.expected {
			t.Errorf("expected \"%s\", got \"%s\"", test.expected, out)
		}
	}
}

func TestConfigureServiceTypeString(t *testing.T) {
	var stringTests = []struct {
		s        ServiceType
		expected string
	}{
		{Glance, "glance"},
		{Keystone, "keystone"},
	}
	for _, test := range stringTests {
		obj := test.s
		out := obj.String()
		if out != test.expected {
			t.Errorf("expected \"%s\", got \"%s\"", test.expected, out)
		}
	}
}
