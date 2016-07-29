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
	"fmt"
	"sync/atomic"

	"github.com/01org/ciao/ssntp/uuid"
)

// NoopDriver is a driver which does nothing.
type NoopDriver struct {
	deviceNum int64
}

// CreateBlockDevice pretends to create a block device.
func (d *NoopDriver) CreateBlockDevice(image *string, size int) (BlockDevice, error) {
	return BlockDevice{ID: uuid.Generate().String()}, nil
}

// DeleteBlockDevice pretends to delete a block device.
func (d *NoopDriver) DeleteBlockDevice(string) error {
	return nil
}

// MapVolumeToNode pretends to map a volume to a local device on a node.
func (d *NoopDriver) MapVolumeToNode(volumeUUID string) (string, error) {
	dNum := atomic.AddInt64(&d.deviceNum, 1)
	return fmt.Sprintf("/dev/blk%d", dNum), nil
}

// UnmapVolumeFromNode pretends to unmap a volume from a local device on a node.
func (d *NoopDriver) UnmapVolumeFromNode(devNmae string) error {
	return nil
}

// GetVolumeMapping returns an empty slice, indicating no devices are mapped to the
// specified volume.
func (d *NoopDriver) GetVolumeMapping() (map[string][]string, error) {
	return nil, nil
}
