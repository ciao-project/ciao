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
	"strconv"
	"testing"

	. "github.com/01org/ciao/payloads"
	"github.com/01org/ciao/testutil"
	"gopkg.in/yaml.v2"
)

func TestConfigureUnmarshal(t *testing.T) {
	var cfg Configure

	err := yaml.Unmarshal([]byte(testutil.ConfigureYaml), &cfg)
	if err != nil {
		t.Error(err)
	}

	if cfg.Configure.ImageService.Type != Glance {
		t.Errorf("Wrong image service type [%s]", cfg.Configure.ImageService.Type)
	}

	if cfg.Configure.IdentityService.Type != Keystone {
		t.Errorf("Wrong identity service type [%s]", cfg.Configure.IdentityService.Type)
	}

	if cfg.Configure.Launcher.ManagementNetwork != testutil.MgmtNet {
		t.Errorf("Wrong launcher management network [%s]", cfg.Configure.Launcher.ManagementNetwork)
	}

	if cfg.Configure.Launcher.ComputeNetwork != testutil.ComputeNet {
		t.Errorf("Wrong launcher compute network [%s]", cfg.Configure.Launcher.ComputeNetwork)
	}

	if cfg.Configure.Scheduler.ConfigStorageType != Filesystem {
		t.Errorf("Wrong scheduler storage type [%s]", cfg.Configure.Scheduler.ConfigStorageType)
	}

	p, _ := strconv.Atoi(testutil.ComputePort)
	if cfg.Configure.Controller.ComputePort != p {
		t.Errorf("Wrong controller compute port [%d]", cfg.Configure.Controller.ComputePort)
	}
}

func TestConfigureMarshal(t *testing.T) {
	var cfg Configure

	cfg.Configure.ImageService.Type = Glance
	cfg.Configure.ImageService.URL = testutil.GlanceURL

	cfg.Configure.IdentityService.Type = Keystone
	cfg.Configure.IdentityService.URL = testutil.KeystoneURL

	cfg.Configure.Launcher.ComputeNetwork = testutil.ComputeNet
	cfg.Configure.Launcher.ManagementNetwork = testutil.MgmtNet
	cfg.Configure.Launcher.DiskLimit = false
	cfg.Configure.Launcher.MemoryLimit = false

	p, _ := strconv.Atoi(testutil.ComputePort)
	cfg.Configure.Controller.ComputePort = p
	cfg.Configure.Controller.HTTPSCACert = testutil.HTTPSCACert
	cfg.Configure.Controller.HTTPSKey = testutil.HTTPSKey
	cfg.Configure.Controller.IdentityUser = testutil.IdentityUser
	cfg.Configure.Controller.IdentityPassword = testutil.IdentityPassword

	cfg.Configure.Scheduler.ConfigStorageType = Filesystem
	cfg.Configure.Scheduler.ConfigStorageURI = testutil.StorageURI

	y, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.ConfigureYaml {
		t.Errorf("CONFIGURE marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.ConfigureYaml)
	}
}

func TestConfigureStorageTypeString(t *testing.T) {
	var stringTests = []struct {
		s        StorageType
		expected string
	}{
		{Filesystem, Filesystem.String()},
		{Etcd, Etcd.String()},
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
		{Glance, Glance.String()},
		{Keystone, Keystone.String()},
	}
	for _, test := range stringTests {
		obj := test.s
		out := obj.String()
		if out != test.expected {
			t.Errorf("expected \"%s\", got \"%s\"", test.expected, out)
		}
	}
}
