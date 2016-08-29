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
	"testing"
)

func TestGetDistro(t *testing.T) {
	if pathExists("/usr/share/clear/bundles") == false {
		t.Skip("Unsupported test distro")
	}
	d := getDistro()
	if d == nil {
		t.Fatal("Cannot get known distro object")
	}
	if d.getID() == "" {
		t.Fatal("Invalid ID for distro")
	}
}
