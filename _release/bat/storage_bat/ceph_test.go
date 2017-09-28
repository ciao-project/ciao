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

package storagebat

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ciao-project/ciao/bat"
	"github.com/ciao-project/ciao/ciao-storage"
	"gopkg.in/yaml.v2"
)

var driver storage.CephDriver

type storageConfig struct {
	CephID string `yaml:"ceph_id"`
}

type configuration struct {
	Configure struct {
		Storage storageConfig `yaml:"storage"`
	} `yaml:"configure"`
}

// Check creating a ceph backed block device works
//
// TestCreateBlockDevice creates a block device containing some random data,
// checks for errors and then deletes it.
func TestCreateBlockDevice(t *testing.T) {
	if driver.ID == "" {
		t.Skip("Skipping test: Ceph ID not set")
	}

	path, err := bat.CreateRandomFile(20)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	device, err := driver.CreateBlockDevice("", path, 0)
	if err != nil {
		t.Fatal(err)
	}

	err = driver.DeleteBlockDevice(device.ID)
	if err != nil {
		t.Fatal(err)
	}
}

// Test creating a sized ceph backed block device.
//
// TestCreateSizedBlockDevice creates a block device of a fixed size, checking
// for errors and then checks that the size is a expected.
func TestCreateSizedBlockDevice(t *testing.T) {
	if driver.ID == "" {
		t.Skip("Skipping test: Ceph ID not set")
	}

	device, err := driver.CreateBlockDevice("", "", 1)
	if err != nil {
		t.Fatal(err)
	}

	blockSize, err := driver.GetBlockDeviceSize(device.ID)
	if err != nil {
		t.Fatal(err)
	}

	if blockSize != 1*1024*1024*1024 {
		t.Fatalf("Unexpected block size (%v): expected: %v got: %v", device.ID, 1*1024*1024*1024, blockSize)
	}

	err = driver.DeleteBlockDevice(device.ID)
	if err != nil {
		t.Fatal(err)
	}
}

// Check copying a ceph backed block device works
//
// TestCopyBlockDevice creates a block device containing some random data,
// checks for errors and then copies it. The created volumes are then
// deleted.
func TestCopyBlockDevice(t *testing.T) {
	if driver.ID == "" {
		t.Skip("Skipping test: Ceph ID not set")
	}

	path, err := bat.CreateRandomFile(20)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	device, err := driver.CreateBlockDevice("", path, 0)
	if err != nil {
		t.Fatal(err)
	}

	copy, err := driver.CopyBlockDevice(device.ID)
	if err != nil {
		t.Fatal(err)
	}

	err = driver.DeleteBlockDevice(copy.ID)
	if err != nil {
		t.Fatal(err)
	}

	err = driver.DeleteBlockDevice(device.ID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMain(m *testing.M) {
	var config configuration
	data, err := ioutil.ReadFile("/etc/ciao/configuration.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config file: %v", err)
	} else {
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing config file: %v", err)
		} else {
			driver.ID = config.Configure.Storage.CephID
		}
	}

	code := m.Run()
	os.Exit(code)
}
