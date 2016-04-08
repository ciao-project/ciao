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
	"net"
	"testing"

	"github.com/01org/ciao/networking/libsnnet"
)

//Test all GRE tunnel primitives
//
//Tests create, enable, disable and destroy of GRE tunnels
//Failure indicates changes in netlink or kernel and in some
//case pre-existing tunnels on the test node. Ensure that
//there are no existing conflicting tunnels before running
//this test
//
//Test is expected to pass
func TestGre_Basic(t *testing.T) {
	id := "testgretap"
	local := net.ParseIP("127.0.0.1")
	remote := local
	key := uint32(0xF)

	gre, _ := libsnnet.NewGreTunEP(id, local, remote, key)

	if err := gre.Create(); err != nil {
		t.Errorf("GreTunnel creation failed: %v", err)
	}

	if err := gre.Enable(); err != nil {
		t.Errorf("GreTunnel enable failed: %v", err)
	}

	if err := gre.Disable(); err != nil {
		t.Errorf("GreTunnel disable failed: %v", err)
	}

	if err := gre.Destroy(); err != nil {
		t.Errorf("GreTunnel deletion failed: %v", err)
	}
}

//Test GRE tunnel bridge interactions
//
//Test all bridge, gre tunnel interactions including
//attach, detach, enable, disable, destroy
//
//Test is expected to pass
func TestGre_Bridge(t *testing.T) {
	id := "testgretap"
	local := net.ParseIP("127.0.0.1")
	remote := local
	key := uint32(0xF)

	gre, _ := libsnnet.NewGreTunEP(id, local, remote, key)
	bridge, _ := libsnnet.NewBridge("testbridge")

	if err := gre.Create(); err != nil {
		t.Errorf("Vnic Create failed: %v", err)
	}
	defer gre.Destroy()

	if err := bridge.Create(); err != nil {
		t.Errorf("Bridge Create failed: %v", err)
	}
	defer bridge.Destroy()

	if err := gre.Attach(bridge); err != nil {
		t.Errorf("GRE attach failed: %v", err)
	}

	if err := gre.Enable(); err != nil {
		t.Errorf("GRE enable failed: %v", err)
	}

	if err := bridge.Enable(); err != nil {
		t.Errorf("Bridge enable failed: %v", err)
	}

	if err := gre.Detach(bridge); err != nil {
		t.Errorf("GRE detach failed: %v", err)
	}
}
