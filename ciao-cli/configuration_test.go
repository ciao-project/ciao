//
// Copyright (c) 2017 Intel Corporation
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
package main

import "testing"

var configURICases = []struct {
	uri      string // URI to evaluate
	scheme   string // expected case to handle validation
	expected bool   // expected result of the test case
}{
	{"http://identity.example.com:35357", "http", false},   // invalid URI (insecure scheme)
	{"http://identity.example.com:35357", "https", false},  // invalid URI (invalid scheme)
	{"https://:8080/subdir", "https", false},               // invalid URI (no host)
	{"file://no-path-defined.example.com", "file", false},  // invalid URI (no path)
	{"https://some-host.example.com:35357", "https", true}, // valid URI (host defined)
	{"file:///etc/ciao/configuration.yaml", "file", true},  // valid URI (path defined)
}

var configBooleanCases = []struct {
	boolStr  string // boolean string to evaluate
	saneBool string // boolean value sanatized
	expected bool   // expected result of the test case
}{
	{"1", "true", true},
	{"t", "true", true},
	{"T", "true", true},
	{"true", "true", true},
	{"TruE", "true", true},
	{"TRUE", "true", true},
	{"0", "false", true},
	{"f", "false", true},
	{"F", "false", true},
	{"false", "false", true},
	{"FalsE", "false", true},
	{"FALSE", "false", true},
	{"invalid", "", false},
}

var configNumberCases = []struct {
	numStr   string // number string to evaluate
	expected bool   // expected result of the test case
}{
	{"45", true},
	{"invalid", false},
}

var configValueCases = []struct {
	confElement string // element of configuration to test
	confValue   string // value of the element in the configuration to test
	expected    bool   // expected result of the test case
}{
	{"scheduler.storage_uri", "file:///etc/ciao/configuration.yaml", true},     // valid location for config
	{"scheduler.storage_uri", "https:///etc/ciao/configuration.yaml", false},   // invalid scheme for config
	{"storage.ceph_id", "42", false},                                           // unsupported configurable element
	{"storage.ceph_id", "", false},                                             // unsupported configurable element
	{"controller.volume_port", "42", false},                                    // unsupported configurable element
	{"controller.volume_port", "NaN", false},                                   // unsupported configurable element
	{"controller.compute_port", "8774", true},                                  // valid configurable value
	{"controller.compute_port", "NaN", false},                                  // invalid port
	{"controller.ciao_port", "8889", false},                                    // unsupported configurable element
	{"controller.ciao_port", "NaN", false},                                     // unsupported configurable element
	{"controller.compute_fqdn", "invalid", false},                              // unsupported configurable element
	{"controller.compute_fqdn", "compute.example.com", false},                  // unsupported configurable element
	{"controller.compute_ca", "notACert", false},                               // unsupported configurable element
	{"controller.compute_ca", "/etc/pki/ciao/api/controller_cert.pem", false},  // unsupported configurable element
	{"controller.compute_cert", "notACert", false},                             // unsupported configurable element
	{"controller.compute_cert", "/etc/pki/ciao/api/controller_key.pem", false}, // unsupported configurable element
	{"controller.identity_user", "", false},                                    // unsupported configurable element
	{"controller.identity_user", "admin", false},                               // unsupported configurable element
	{"controller.identity_password", "", false},                                // unsupported configurable element
	{"controller.identity_password", "someAdminPassword", false},               // unsupported configurable element
	{"controller.cnci_vcpus", "0", false},                                      // unsupported configurable element
	{"controller.cnci_vcpus", "NaN", false},                                    // unsupported configurable element
	{"controller.cnci_mem", "0", false},                                        // unsupported configurable element
	{"controller.cnci_mem", "NaN", false},                                      // unsupported configurable element
	{"controller.cnci_disk", "0", false},                                       // unsupported configurable element
	{"controller.cnci_disk", "NaN", false},                                     // unsupported configurable element
	{"controller.admin_ssh_key", "", false},                                    // unsupported configurable element
	{"controller.admin_ssh_password", "notAKey", false},                        // unsupported configurable element
	{"launcher.compute_net", "- 192.138.0.1/24", false},                        // unsupported configurable element
	{"launcher.compute_net", "notANetList", false},                             // unsupported configurable element
	{"launcher.mgmt_net", "- 192.138.0.1/24", false},                           // unsupported configurable element
	{"launcher.mgmt_net", "notANetList", false},                                // unsupported configurable element
	{"launcher.disk_limit", "true", true},                                      // valid value for disk_limit
	{"launcher.disk_limit", "false", true},                                     // valid value for disk_limit
	{"launcher.disk_limit", "invalid", false},                                  // invalid value for disk_limit
	{"launcher.mem_limit", "true", true},                                       // valid value for mem_limit
	{"launcher.mem_limit", "false", true},                                      // valid value for mem_limit
	{"launcher.mem_limit", "invalid", false},                                   // invalid value for mem_limit
	{"image_service.type", "glance", false},                                    // unsupported configurable element
	{"image_service.type", "non-glance", false},                                // unsupported configurable element
	{"image_service.url", "https://image.example.com", false},                  // unsupported configurable element
	{"image_service.url", "invalidURL", false},                                 // unsupported configurable element
	{"identity_service.type", "keystone", true},                                // valid value for identity_service.type
	{"identity_service.type", "non-keystone", false},                           // unsupported configurable element
	{"identity_service.url", "https://keystone.example.com:35357", true},       // valid value for identity_service.url
	{"identity_service.url", "https://keystone.example.com:35358", true},       // valid value for identity_service.url
	{"identity_service.url", "invalidURL", false},                              // invalid value for identity_service.url
}

func testValidConfigURICase(t *testing.T, uri string, scheme string, pass bool) {
	val := validConfigURI(uri, scheme)
	if (pass == false && val == true) || (pass == true && val == false) {
		t.Fatalf("expected %v, got %v", pass, val)
	}
}

// TestConfigurationValidConfigURI tests the 'validConfigURI' function
// This test is expected to pass
func TestConfigurationValidConfigURI(t *testing.T) {
	for _, tt := range configURICases {
		testValidConfigURICase(t, tt.uri, tt.scheme, tt.expected)
	}

}

// TestConfigurationValidConfigBoolean tests the 'validConfigBoolean' function
// This test is expected to pass
func TestConfigurationValidConfigBoolean(t *testing.T) {
	for _, tt := range configBooleanCases {
		ret := validConfigBoolean(tt.boolStr)
		if ret != tt.expected {
			t.Fatalf("expected %v, got %v", tt.expected, ret)
		}
	}
}

// TestConfigurationValidConfigBoolean tests the 'sanatizeBoolean' function
// This test is expected to pass
func TestConfigurationSanatizeBoolean(t *testing.T) {
	for _, tt := range configBooleanCases {
		ret := sanatizeBoolean(tt.boolStr)
		if ret != tt.saneBool {
			t.Fatalf("expected %v, got %v", tt.saneBool, ret)
		}
	}
}

// TestConfigurationValidConfigNumber tests the 'validConfigNumber' function
// This test is expected to pass
func TestConfigurationValidConfigNumber(t *testing.T) {
	for _, tt := range configNumberCases {
		ret := validConfigNumber(tt.numStr)
		if ret != tt.expected {
			t.Fatalf("expected %v, got %v", tt.expected, ret)
		}
	}
}

// TestConfigurationValidConfigValue tests the 'validConfigValue' function
// This test is expected to pass
func TestConfigurationValidConfigValue(t *testing.T) {
	for _, tt := range configValueCases {
		ret := validConfigValue(tt.confValue, tt.confElement)
		if ret != tt.expected {
			t.Fatalf("expected %v, got %v", tt.expected, ret)
		}
	}
}
