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

package datastore

import (
	"io"
	"os"
	"path"
)

// Posix implements the DataStore interface for posix filesystems
type Posix struct {
	MountPoint string
}

// Write copies an image into the posix filesystem.
// If the image already exists it will be overridden.
func (p *Posix) Write(ID string, body io.Reader) (int64, error) {
	imageName := path.Join(p.MountPoint, ID)

	image, err := os.Create(imageName)
	if err != nil {
		return 0, err
	}

	buf := make([]byte, 1<<16)

	return io.CopyBuffer(image, body, buf)
}

// Delete removes an image from the posix filesystem
func (p *Posix) Delete(ID string) error {
	imageName := path.Join(p.MountPoint, ID)

	return os.Remove(imageName)
}
