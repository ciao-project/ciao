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
	"testing"
)

func TestCreateBlockDevice(t *testing.T) {
	driver := cephDriver{
		SecretPath: "/etc/ceph/ceph.client.kristen.keyring",
		ID:         "kristen",
	}

	imagePath := "/var/lib/ciao/images/73a86d7e-93c0-480e-9c41-ab42f69b7799"

	device, err := driver.CreateBlockDevice(&imagePath, 0)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(device.ID)
}
