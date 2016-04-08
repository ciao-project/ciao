//
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
//

package libsnnet_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/01org/ciao/networking/libsnnet"
)

var scaleCfg = struct {
	maxBridgesShort int
	maxVnicsShort   int
	maxBridgesLong  int
	maxVnicsLong    int
}{2, 64, 8, 64}

//Internal scaling test case
//
//Test used to determine how many interfaces can be created
//on any given node. This tests the underlying kernel and node
//configuration and not the primitive itself
//
//Test may fail or take a long time to run based on the values
//configured for maX*
func TestScale(t *testing.T) {

	var maxBridges, maxVnics int
	if testing.Short() {
		maxBridges = scaleCfg.maxBridgesShort
		maxVnics = scaleCfg.maxVnicsShort
	} else {
		maxBridges = scaleCfg.maxBridgesLong
		maxVnics = scaleCfg.maxVnicsLong
	}

	unique := false

	if os.Getenv("UNIQUE") != "" {
		unique = true
		t.Logf("Uniqueness test on")
	} else {
		t.Logf("Uniqueness test off")
	}

	for b := 0; b < maxBridges; b++ {
		var err error

		bridge, _ := libsnnet.NewBridge(fmt.Sprintf("testbridge%v", b))

		if bridge.LinkName, err = libsnnet.GenIface(bridge, unique); err != nil {
			t.Errorf("Bridge Interface generation failed: %v %v", err, bridge)
		}

		if err := bridge.Create(); err != nil {
			t.Errorf("Bridge create failed: %v %v", err, bridge)
		}
		defer bridge.Destroy()

		for v := 0; v < maxVnics; v++ {
			vnic, _ := libsnnet.NewVnic(fmt.Sprintf("testvnic%v_%v", v, b))
			if vnic.LinkName, err = libsnnet.GenIface(vnic, unique); err != nil {
				t.Errorf("VNIC Interface generation failed: %v %v", err, bridge)
			}

			if err := vnic.Create(); err != nil {
				t.Errorf("Vnic Create failed: %v %v", err, vnic)
			}

			defer vnic.Destroy()

			if err := vnic.Attach(bridge); err != nil {
				t.Errorf("Vnic attach failed: %v", err)
			}
			if err := vnic.Enable(); err != nil {
				t.Errorf("Vnic enable failed: %v", err)
			}

			defer vnic.Detach(bridge)

		}
		if err := bridge.Enable(); err != nil {
			t.Errorf("Vnic enable failed: %v", err)
		}
	}
}
