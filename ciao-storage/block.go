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
	"errors"
)

var (
	// ErrNoDevice is returned from a driver
	ErrNoDevice = errors.New("Not able to create device")
)

// BlockDriver is the interface that all block drivers must implement.
type BlockDriver interface {
	CreateBlockDevice(image *string, sizeGB int) (BlockDevice, error)
	DeleteBlockDevice(string) error
	MapVolumeToNode(volumeUUID string) (string, error)
	UnmapVolumeFromNode(devName string) error
	GetVolumeMapping() (map[string][]string, error)
}

// BlockDevice contains information about a block devices.
type BlockDevice struct {
	ID string
}
