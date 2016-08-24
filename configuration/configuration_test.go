//
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
//

package configuration

import (
	"bytes"
	"io/ioutil"
	"syscall"
	"testing"

	"github.com/01org/ciao/networking/libsnnet"
	"github.com/01org/ciao/payloads"
	"github.com/google/gofuzz"
)

const badScheme = "badScheme://non-existent/path/nofile"
const invalidURI = "file://%z invalid uri with spaces"
const emptyPathURI = "file://"

const keystoneURL = "http://keystone.example.com"
const glanceURL = "http://glance.example.com"
const computeNet = "192.168.1.0/24"
const mgmtNet = "192.168.1.0/24"
const storageURI = "/etc/ciao/configuration.yaml"
const identityUser = "controller"
const identityPassword = "ciao"
const httpsKey = "/etc/pki/ciao/compute_key.pem"
const httpsCACert = "/etc/pki/ciao/compute_ca.pem"
const secretPath = "/etc/ceph/ceph.client.ciao.keyring"
const cephID = "ciao"

const minValidConf = `configure:
  scheduler:
    storage_uri: /etc/ciao/configuration.yaml
  controller:
    compute_ca: /etc/pki/ciao/compute_ca.pem
    compute_cert: /etc/pki/ciao/compute_key.pem
    identity_user: controller
    identity_password: ciao
  launcher:
    compute_net:
    - 192.168.1.0/24
    mgmt_net:
    - 192.168.1.0/24
  identity_service:
    url: http://keystone.example.com
`
const fullValidConf = `configure:
  scheduler:
    storage_type: file
    storage_uri: /etc/ciao/configuration.yaml
  storage:
    secret_path: /etc/ceph/ceph.client.ciao.keyring
    ceph_id: ciao
  controller:
    compute_port: 8774
    compute_ca: /etc/pki/ciao/compute_ca.pem
    compute_cert: /etc/pki/ciao/compute_key.pem
    identity_user: controller
    identity_password: ciao
  launcher:
    compute_net:
    - 192.168.1.0/24
    mgmt_net:
    - 192.168.1.0/24
    disk_limit: true
    mem_limit: true
  image_service:
    type: glance
    url: http://glance.example.com
  identity_service:
    type: keystone
    url: http://keystone.example.com
`

func testBlob(t *testing.T, conf *payloads.Configure, expectedBlob []byte, positive bool) {
	blob, err := Blob(conf)
	if positive == true && err != nil {
		t.Fatalf("%s", err)
	}
	if positive == false && err == nil {
		t.Fatalf("%s", err)
	}
	if positive == true && expectedBlob != nil {
		if bytes.Equal(expectedBlob, blob) == false {
			t.Fatalf("expected %v got %v", expectedBlob, blob)
		}
	}
}

func TestBlobEmptyPayload(t *testing.T) {
	testBlob(t, &payloads.Configure{}, nil, false)
}

func TestBlobCorrectPayload(t *testing.T) {
	var payload payloads.Configure
	fillPayload(&payload)
	testBlob(t, &payload, []byte(fullValidConf), true)
}

func equalPayload(p1, p2 payloads.Configure) bool {
	return (p1.Configure.Scheduler.ConfigStorageType == p2.Configure.Scheduler.ConfigStorageType &&
		p1.Configure.Scheduler.ConfigStorageURI == p2.Configure.Scheduler.ConfigStorageURI &&

		p1.Configure.Controller.ComputePort == p2.Configure.Controller.ComputePort &&
		p1.Configure.Controller.HTTPSCACert == p2.Configure.Controller.HTTPSCACert &&
		p1.Configure.Controller.HTTPSKey == p2.Configure.Controller.HTTPSKey &&
		p1.Configure.Controller.IdentityUser == p2.Configure.Controller.IdentityUser &&
		p1.Configure.Controller.IdentityPassword == p2.Configure.Controller.IdentityPassword &&

		libsnnet.EqualNetSlice(p1.Configure.Launcher.ComputeNetwork, p2.Configure.Launcher.ComputeNetwork) &&
		libsnnet.EqualNetSlice(p1.Configure.Launcher.ManagementNetwork, p2.Configure.Launcher.ManagementNetwork) &&
		p1.Configure.Launcher.DiskLimit == p2.Configure.Launcher.DiskLimit &&
		p1.Configure.Launcher.MemoryLimit == p2.Configure.Launcher.MemoryLimit &&

		p1.Configure.ImageService.Type == p2.Configure.ImageService.Type &&
		p1.Configure.ImageService.URL == p2.Configure.ImageService.URL &&

		p1.Configure.IdentityService.Type == p2.Configure.IdentityService.Type &&
		p1.Configure.IdentityService.URL == p2.Configure.IdentityService.URL)
}

func emptyPayload(p payloads.Configure) bool {
	return (p.Configure.Scheduler.ConfigStorageType != "" &&
		p.Configure.Scheduler.ConfigStorageURI != "" &&

		p.Configure.Controller.ComputePort != 0 &&
		p.Configure.Controller.HTTPSCACert != "" &&
		p.Configure.Controller.HTTPSKey != "" &&
		p.Configure.Controller.IdentityUser != "" &&
		p.Configure.Controller.IdentityPassword != "" &&

		len(p.Configure.Launcher.ComputeNetwork) > 0 &&
		len(p.Configure.Launcher.ManagementNetwork) > 0 &&
		p.Configure.Launcher.DiskLimit != false &&
		p.Configure.Launcher.MemoryLimit != false &&

		p.Configure.ImageService.Type != "" &&
		p.Configure.ImageService.URL != "" &&

		p.Configure.IdentityService.Type != "" &&
		p.Configure.IdentityService.URL != "")
}

func fillPayload(conf *payloads.Configure) {
	conf.InitDefaults()
	conf.Configure.Scheduler.ConfigStorageURI = storageURI
	conf.Configure.Controller.HTTPSCACert = httpsCACert
	conf.Configure.Controller.HTTPSKey = httpsKey
	conf.Configure.Controller.IdentityUser = identityUser
	conf.Configure.Controller.IdentityPassword = identityPassword
	conf.Configure.Launcher.ComputeNetwork = []string{computeNet}
	conf.Configure.Launcher.ManagementNetwork = []string{mgmtNet}
	conf.Configure.ImageService.URL = glanceURL
	conf.Configure.IdentityService.URL = keystoneURL
	conf.Configure.Storage.SecretPath = secretPath
	conf.Configure.Storage.CephID = cephID
}

func testPayload(t *testing.T, blob []byte, expectedConf payloads.Configure, positive bool) {
	conf, err := Payload(blob)

	// expected FAIL
	if positive == false && err != nil {
		// do nothing...expected case.
	}

	// unexpected FAIL
	if positive == true && err != nil {
		t.Fatalf("%s", err)
	}

	// unexpected FAIL
	if positive == true && emptyPayload(expectedConf) == false {
		if equalPayload(expectedConf, conf) == false {
			t.Fatalf("Expected %v got %v", expectedConf, conf)
		}
	}
}

func TestPayloadNilBlob(t *testing.T) {
	var conf payloads.Configure
	testPayload(t, nil, conf, false)
}

func TestPayloadFuzzyBlob(t *testing.T) {
	var conf payloads.Configure
	var fuzzyStr string
	f := fuzz.New()
	f.Fuzz(&fuzzyStr)
	testPayload(t, []byte(fuzzyStr), conf, false)
}

func TestPayloadCorrectBlob(t *testing.T) {
	var expectedConf payloads.Configure

	fillPayload(&expectedConf)
	testPayload(t, []byte(fullValidConf), expectedConf, true)
}

func saneDefaults(conf *payloads.Configure) bool {
	return (conf.Configure.Scheduler.ConfigStorageType == payloads.Filesystem &&
		conf.Configure.Controller.ComputePort == 8774 &&
		conf.Configure.ImageService.Type == payloads.Glance &&
		conf.Configure.IdentityService.Type == payloads.Keystone &&
		conf.Configure.Launcher.DiskLimit == true &&
		conf.Configure.Launcher.MemoryLimit == true)
}

func TestInitDefaults(t *testing.T) {
	var conf payloads.Configure
	conf.InitDefaults()
	res := saneDefaults(&conf)
	if res != true {
		t.Fatalf("Expected true, got %v", res)
	}
}

func TestValidMinConf(t *testing.T) {
	var conf payloads.Configure

	conf.Configure.Scheduler.ConfigStorageURI = storageURI
	conf.Configure.Controller.HTTPSCACert = httpsCACert
	conf.Configure.Controller.HTTPSKey = httpsKey
	conf.Configure.Controller.IdentityUser = identityUser
	conf.Configure.Controller.IdentityPassword = identityPassword
	conf.Configure.Launcher.ComputeNetwork = []string{computeNet}
	conf.Configure.Launcher.ManagementNetwork = []string{mgmtNet}

	valid := validMinConf(&conf)
	if valid != false {
		t.Fatalf("Expected false, got %v", valid)
	}

	// missing value to get minimal valid Configuration
	conf.Configure.IdentityService.URL = keystoneURL

	valid = validMinConf(&conf)
	if valid != true {
		t.Fatalf("Expected true, got %v", valid)
	}
}

func testExtractBlob(t *testing.T, uri string, expectedBlob []byte, positive bool) {
	blob, err := ExtractBlob(uri)
	// expected FAIL
	if positive == false && err == nil {
		t.Fatalf("%s", err)
	}
	// expected PASS
	if positive == true && err != nil {
		t.Fatalf("%s", err)
	}
	if positive == true && expectedBlob != nil {
		if bytes.Equal(expectedBlob, blob) == false {
			t.Fatalf("Expected %v got %v", expectedBlob, blob)
		}
	}
}

func TestExtractBlobInvalidURI(t *testing.T) {
	testExtractBlob(t, invalidURI, nil, false)
}

func TestExtractBlobEmptyPathURI(t *testing.T) {
	testExtractBlob(t, emptyPathURI, nil, false)
}

func TestExtractBlobBadSchemeURI(t *testing.T) {
	testExtractBlob(t, badScheme, nil, false)
}

func TestExtractBlobMalformedConf(t *testing.T) {
	// create temp file where we can read the conf
	tmpf, err := ioutil.TempFile("", "configuration.yaml")
	if err != nil {
		panic(err)
	}
	defer syscall.Unlink(tmpf.Name())
	ioutil.WriteFile(tmpf.Name(), []byte(malformedConf), 0644)

	testExtractBlob(t, "file://"+tmpf.Name(), nil, false)
}

func TestExtractBlobValidConf(t *testing.T) {
	// create temp file where we can read the conf
	tmpf, err := ioutil.TempFile("", "configuration.yaml")
	if err != nil {
		panic(err)
	}
	defer syscall.Unlink(tmpf.Name())
	ioutil.WriteFile(tmpf.Name(), []byte(fullValidConf), 0644)

	testExtractBlob(t, "file://"+tmpf.Name(), []byte(fullValidConf), true)
}
