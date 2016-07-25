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
	"os/exec"

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
	if imagePath == nil {
		// we'd need to have more details about what kind
		// of device to create. So, we need to change
		// the API to support this - with some opts.
		return BlockDevice{}, ErrNoDevice
	}

	// generate a UUID to use for this image.
	ID := uuid.Generate().String()

	cmd := exec.Command("rbd", "--keyring", d.SecretPath, "--id", d.ID, "--image-format", "2", "import", *imagePath, ID)

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
