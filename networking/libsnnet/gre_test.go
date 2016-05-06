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

package libsnnet

import (
	"net"
	"testing"
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

	gre, _ := newGreTunEP(id, local, remote, key)

	if err := gre.create(); err != nil {
		t.Errorf("GreTunnel creation failed: %v", err)
	}

	if err := gre.enable(); err != nil {
		t.Errorf("GreTunnel enable failed: %v", err)
	}

	if err := gre.getDevice(); err != nil {
		t.Errorf("GreTunnel enable failed: %v", err)
	}

	if err := gre.disable(); err != nil {
		t.Errorf("GreTunnel disable failed: %v", err)
	}

	if err := gre.destroy(); err != nil {
		t.Errorf("GreTunnel deletion failed: %v", err)
	}

	if err := gre.destroy(); err == nil {
		t.Errorf("GreTunnel deletion should have failed")
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

	gre, _ := newGreTunEP(id, local, remote, key)
	bridge, _ := newBridge("testbridge")

	if err := gre.create(); err != nil {
		t.Errorf("Vnic Create failed: %v", err)
	}
	defer func() { _ = gre.destroy() }()

	if err := bridge.create(); err != nil {
		t.Errorf("Bridge Create failed: %v", err)
	}
	defer func() { _ = bridge.destroy() }()

	if err := gre.attach(bridge); err != nil {
		t.Errorf("GRE attach failed: %v", err)
	}
	//Duplicate
	if err := gre.attach(bridge); err != nil {
		t.Errorf("GRE attach failed: %v", err)
	}

	if err := gre.enable(); err != nil {
		t.Errorf("GRE enable failed: %v", err)
	}

	if err := bridge.enable(); err != nil {
		t.Errorf("Bridge enable failed: %v", err)
	}

	if err := gre.detach(bridge); err != nil {
		t.Errorf("GRE detach failed: %v", err)
	}

	//Duplicate
	if err := gre.detach(bridge); err != nil {
		t.Errorf("GRE detach failed: %v", err)
	}

}

//Tests failure paths in the GRE tunnel
//
//Tests failure paths in the GRE tunnel
//
//Test is expected to pass
func TestGre_Negative(t *testing.T) {
	id := "testgretap"
	local := net.ParseIP("127.0.0.1")
	remote := local
	key := uint32(0xF)

	gre, _ := newGreTunEP(id, local, remote, key)
	greDupl, _ := newGreTunEP(id, local, remote, key)

	if err := gre.create(); err != nil {
		t.Errorf("GreTunnel creation failed: %v", err)
	}

	if err := greDupl.create(); err == nil {
		t.Errorf("GreTunnel creation should have failed")
	}
	if err := greDupl.enable(); err == nil {
		t.Errorf("GreTunnel creation should have failed")
	}
	if err := greDupl.disable(); err == nil {
		t.Errorf("GreTunnel creation should have failed")
	}
	if err := greDupl.destroy(); err == nil {
		t.Errorf("GreTunnel creation should have failed")
	}

	if err := gre.enable(); err != nil {
		t.Errorf("GreTunnel enable failed: %v", err)
	}

	if err := gre.disable(); err != nil {
		t.Errorf("GreTunnel disable failed: %v", err)
	}

	if err := gre.destroy(); err != nil {
		t.Errorf("GreTunnel deletion failed: %v", err)
	}
}
