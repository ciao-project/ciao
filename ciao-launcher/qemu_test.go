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

package main

import (
	"bytes"
	"reflect"
	"testing"
)

var imageInfoTestGood = `
image: /var/lib/ciao/images/clear-8780-cloud.img
file format: qcow2
virtual size: 865M (907018240 bytes)
disk size: 113M
cluster_size: 65536
Format specific information:
    compat: 1.1
    lazy refcounts: false
    refcount bits: 16
    corrupt: false
`

var imageInfoTestMissingBytes = `
image: /var/lib/ciao/images/clear-8780-cloud.img
file format: qcow2
virtual size: 865M
disk size: 113M
cluster_size: 65536
Format specific information:
    compat: 1.1
    lazy refcounts: false
    refcount bits: 16
    corrupt: false
`

var imageInfoTestMissingLine = `
image: /var/lib/ciao/images/clear-8780-cloud.img
file format: qcow2
disk size: 113M
cluster_size: 65536
Format specific information:
    compat: 1.1
    lazy refcounts: false
    refcount bits: 16
    corrupt: false
`

var imageInfoTooBig = `
image: /var/lib/ciao/images/clear-8780-cloud.img
file format: qcow2
virtual size: 18,446,744,073,710M (18446744073709551615 bytes)
disk size: 113M
cluster_size: 65536
Format specific information:
    compat: 1.1
    lazy refcounts: false
    refcount bits: 16
    corrupt: false
`

var imageInfoBadBytes = `
image: /var/lib/ciao/images/clear-8780-cloud.img
file format: qcow2
virtual size: 865M (9aaaa07018240 bytes)
disk size: 113M
cluster_size: 65536
Format specific information:
    compat: 1.1
    lazy refcounts: false
    refcount bits: 16
    corrupt: false
`

func TestExtractImageInfo(t *testing.T) {
	tests := []struct {
		name   string
		result int
		data   string
	}{
		{
			"imageInfoTestGood",
			908,
			imageInfoTestGood,
		},
		{
			"imageInfoTestMissingBytes",
			-1,
			imageInfoTestMissingBytes,
		},
		{
			"imageInfoTestMissingLine",
			-1,
			imageInfoTestMissingLine,
		},
		{
			"imageInfoTooBig",
			-1,
			imageInfoTooBig,
		},
		{
			"imageInfoBadBytes",
			-1,
			imageInfoBadBytes,
		},
	}

	for _, ti := range tests {
		b := bytes.NewBuffer([]byte(ti.data))
		result := extractImageInfo(b)
		if result != ti.result {
			t.Fatalf("%s failed. %d != %d", ti.name, result, ti.result)
		}
	}
}

func genQEMUParams(networkParams []string) []string {
	baseParams := []string{
		"-drive",
		"file=/var/lib/ciao/instance/1/image.qcow2,if=virtio,aio=threads,format=qcow2",
		"-drive",
		"file=/var/lib/ciao/instance/1/seed.iso,if=virtio,media=cdrom",
	}
	baseParams = append(baseParams, networkParams...)
	baseParams = append(baseParams, "-enable-kvm", "-cpu", "host", "-daemonize",
		"-qmp", "unix:/var/lib/ciao/instance/1/socket,server,nowait")

	return baseParams
}

func TestGenerateQEMULaunchParams(t *testing.T) {
	var cfg vmConfig

	params := genQEMUParams(nil)
	cfg.Legacy = true
	genParams := generateQEMULaunchParams(&cfg, "/var/lib/ciao/instance/1/seed.iso",
		"/var/lib/ciao/instance/1", nil)
	if !reflect.DeepEqual(params, genParams) {
		t.Fatalf("%s and %s do not match", params, genParams)
	}

	params = genQEMUParams(nil)
	cfg.Legacy = false
	cfg.Mem = 0
	cfg.Cpus = 0
	params = append(params, "-bios", qemuEfiFw)
	genParams = generateQEMULaunchParams(&cfg, "/var/lib/ciao/instance/1/seed.iso",
		"/var/lib/ciao/instance/1", nil)
	if !reflect.DeepEqual(params, genParams) {
		t.Fatalf("%s and %s do not match", params, genParams)
	}

	params = genQEMUParams(nil)
	cfg.Mem = 100
	cfg.Cpus = 0
	cfg.Legacy = true
	params = append(params, "-m", "100")
	genParams = generateQEMULaunchParams(&cfg, "/var/lib/ciao/instance/1/seed.iso",
		"/var/lib/ciao/instance/1", nil)
	if !reflect.DeepEqual(params, genParams) {
		t.Fatalf("%s and %s do not match", params, genParams)
	}

	params = genQEMUParams(nil)
	cfg.Mem = 0
	cfg.Cpus = 4
	cfg.Legacy = true
	params = append(params, "-smp", "cpus=4")
	genParams = generateQEMULaunchParams(&cfg, "/var/lib/ciao/instance/1/seed.iso",
		"/var/lib/ciao/instance/1", nil)
	if !reflect.DeepEqual(params, genParams) {
		t.Fatalf("%s and %s do not match", params, genParams)
	}

	netParams := []string{"-net", "nic,model=virtio", "-net", "user"}
	params = genQEMUParams(netParams)
	cfg.Mem = 0
	cfg.Cpus = 0
	cfg.Legacy = true
	genParams = generateQEMULaunchParams(&cfg, "/var/lib/ciao/instance/1/seed.iso",
		"/var/lib/ciao/instance/1", netParams)
	if !reflect.DeepEqual(params, genParams) {
		t.Fatalf("%s and %s do not match", params, genParams)
	}
}
