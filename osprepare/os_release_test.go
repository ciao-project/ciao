//
// Copyright © 2016 Intel Corporation
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

package osprepare

import (
	"strings"
	"testing"
)

const (
	NON_EXISTENT_FILE = "/nonexistentpath/this/file/doesnt/exists"
)

func TestGetOsRelease(t *testing.T) {
	d := getDistro()
	if d == nil {
		t.Skip("Unknown distro, cannot test")
	}
	os_rel := GetOsRelease()
	if os_rel == nil {
		t.Fatal("Could not get os-release file for known distro")
	}
	if d.getID() == "clearlinux" && !strings.Contains(os_rel.ID, "clear") {
		t.Fatal("Invalid os-release for clearlinux")
	} else if d.getID() == "ubuntu" && !strings.Contains(os_rel.ID, "ubuntu") {
		t.Fatal("Invalid os-release for Ubuntu")
	} else if d.getID() == "fedora" && !strings.Contains(os_rel.ID, "fedora") {
		t.Fatal("Invalid os-release for Fedora")
	}
}

func TestParseReleaseFileNonExistent(t *testing.T) {
	if res := ParseReleaseFile(NON_EXISTENT_FILE); res != nil {
		t.Fatalf("Expected nil, got %v\n", res)
	}
}
