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

package storage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/01org/ciao/ssntp/uuid"
)

// CephDriver maintains context for the ceph driver interface.
type CephDriver struct {
	// SecretPath is the full path to the cephx keyring file.
	SecretPath string

	// ID is the cephx user ID to use
	ID string
}

// CreateBlockDevice will create a rbd image in the ceph cluster.
func (d CephDriver) CreateBlockDevice(imagePath *string, size int) (BlockDevice, error) {
	// generate a UUID to use for this image.
	ID := uuid.Generate().String()

	var cmd *exec.Cmd

	if imagePath != nil {
		cmd = exec.Command("rbd", "--keyring", d.SecretPath, "--id", d.ID, "--image-format", "2", "import", *imagePath, ID)
	} else {
		// create an empty volume
		cmd = exec.Command("rbd", "--keyring", d.SecretPath, "--id", d.ID, "create", "--size", strconv.Itoa(size), ID)
	}

	_, err := cmd.CombinedOutput()

	if err != nil {
		return BlockDevice{}, err
	}

	return BlockDevice{ID: ID}, nil
}

// DeleteBlockDevice will remove a rbd image from the ceph cluster.
// Not implemented yet.
func (d CephDriver) DeleteBlockDevice(string) error {
	return nil
}

func (d CephDriver) getCredentials() []string {
	args := make([]string, 0, 8)
	if d.SecretPath != "" {
		args = append(args, "--keyring", d.SecretPath)
	}

	if d.ID != "" {
		args = append(args, "--id", d.ID)
	}
	return args
}

// MapVolumeToNode maps a ceph volume to a rbd device on a node.  The
// path to the new device is returned if the mapping succeeds.
func (d CephDriver) MapVolumeToNode(volumeUUID string) (string, error) {
	args := append(d.getCredentials(), "map", volumeUUID)
	cmd := exec.Command("rbd", args...)
	data, err := cmd.Output()
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(data))
	if !scanner.Scan() {
		return "", fmt.Errorf("Unable to determine device name for %s", volumeUUID)
	}
	return scanner.Text(), nil
}

// UnmapVolumeFromNode unmaps a ceph volume from a local device on a node.
func (d CephDriver) UnmapVolumeFromNode(volumeUUID string) error {
	args := append(d.getCredentials(), "unmap", volumeUUID)
	return exec.Command("rbd", args...).Run()
}

// GetVolumeMapping returns a map of volumeUUID to mapped devices.
func (d CephDriver) GetVolumeMapping() (map[string][]string, error) {
	args := append(d.getCredentials(), "showmapped", "--format", "json")
	cmd := exec.Command("rbd", args...)
	data, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	vmap := map[string]struct {
		Name   string `json:"name"`
		Device string `json:"device"`
	}{}
	err = json.Unmarshal([]byte(data), &vmap)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse output from rbd show mapped: %v", err)
	}

	volumeDevMap := make(map[string][]string)

	for _, v := range vmap {
		volumeDevMap[v.Name] = append(volumeDevMap[v.Name], v.Device)
	}

	return volumeDevMap, nil
}
