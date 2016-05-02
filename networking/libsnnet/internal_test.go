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

// Internal tests (whitebox) for libsnnet
package libsnnet

import (
	"net"
	"testing"
)

//Tests the implementation of the db rebuild from aliases
//
//This test uses a mix of primitives and APIs to check
//the reliability of the dbRebuild API
//
//The test is expected to pass
func TestCN_dbRebuild(t *testing.T) {
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")

	vnicCfg := &VnicConfig{
		VnicIP:     net.IPv4(192, 168, 1, 100),
		ConcIP:     net.IPv4(192, 168, 1, 1),
		VnicMAC:    mac,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	cn := &ComputeNode{}
	alias := genCnVnicAliases(vnicCfg)

	bridgeAlias := alias.bridge
	vnicAlias := alias.vnic
	greAlias := alias.gre

	bridge, _ := NewBridge(bridgeAlias)

	if err := bridge.GetDevice(); err != nil {
		// First instance to land, create the bridge and tunnel
		if err := bridge.Create(); err != nil {
			t.Error("Bridge creation failed: ", err)
		}
		defer bridge.Destroy()

		// Create the tunnel to connect to the CNCI
		local := vnicCfg.VnicIP //Fake it for now
		remote := vnicCfg.ConcIP
		subnetKey := vnicCfg.SubnetKey

		gre, _ := NewGreTunEP(greAlias, local, remote, uint32(subnetKey))

		if err := gre.Create(); err != nil {
			t.Error("GRE Tunnel Creation failed: ", err)
		}
		defer gre.Destroy()

		if err := gre.Attach(bridge); err != nil {
			t.Error("GRE Tunnel attach failed: ", err)
		}

	}

	// Create the VNIC for the instance
	vnic, _ := NewVnic(vnicAlias)

	if err := vnic.Create(); err != nil {
		t.Error("Vnic Create failed: ", err)
	}
	defer vnic.Destroy()

	if err := vnic.Attach(bridge); err != nil {
		t.Error("Vnic attach failed: ", err)
	}

	//Add a second vnic
	vnicCfg.VnicIP = net.IPv4(192, 168, 1, 101)
	alias1 := genCnVnicAliases(vnicCfg)
	vnic1, _ := NewVnic(alias1.vnic)

	if err := vnic1.Create(); err != nil {
		t.Error("Vnic Create failed: ", err)
	}
	defer vnic1.Destroy()

	if err := vnic1.Attach(bridge); err != nil {
		t.Error("Vnic attach failed: ", err)
	}

	/* Test negative test cases */
	if err := cn.DbRebuild(nil); err == nil {
		t.Error("cn.dbRebuild should have failed")
	}

	cn.NetworkConfig = &NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          GreTunnel,
	}

	if err := cn.DbRebuild(nil); err == nil {
		t.Error("cn.dbRebuild should have failed")
	}

	/* Test positive */
	cn.cnTopology = &cnTopology{
		bridgeMap: make(map[string]map[string]bool),
		linkMap:   make(map[string]*linkInfo),
		nameMap:   make(map[string]bool),
	}

	if err := cn.DbRebuild(nil); err != nil {
		t.Error("cn.dbRebuild failed", err)
	}

	if cnt, err := cn.dbUpdate(alias.bridge, alias1.vnic, dbDelVnic); err == nil {
		if cnt != 1 {
			t.Error("cn.dbUpdate failed", cnt)
		}
	} else {
		t.Error("cn.dbUpdate failed", err)
	}

	if cnt, err := cn.dbUpdate(alias.bridge, alias.vnic, dbDelVnic); err == nil {
		if cnt != 0 {
			t.Error("cn.dbUpdate failed", cnt)
		}
	} else {
		t.Error("cn.dbUpdate failed", err)
	}

	if cnt, err := cn.dbUpdate(alias.bridge, "", dbDelBr); err == nil {
		if cnt != 0 {
			t.Error("cn.dbUpdate failed", cnt)
		}
	} else {
		t.Error("cn.dbUpdate failed", err)
	}

	if cnt, err := cn.dbUpdate(alias.bridge, "", dbInsBr); err == nil {
		if cnt != 1 {
			t.Error("cn.dbUpdate failed", cnt)
		}
	} else {
		t.Error("cn.dbUpdate failed", err)
	}

	if cnt, err := cn.dbUpdate(alias.bridge, alias.vnic, dbInsVnic); err == nil {
		if cnt != 1 {
			t.Error("cn.dbUpdate failed", cnt)
		}
	} else {
		t.Error("cn.dbUpdate failed", err)
	}

	if cnt, err := cn.dbUpdate(alias.bridge, alias1.vnic, dbInsVnic); err == nil {
		if cnt != 2 {
			t.Error("cn.dbUpdate failed", cnt)
		}
	} else {
		t.Error("cn.dbUpdate failed", err)
	}

	//Negative tests
	if cnt, err := cn.dbUpdate(alias.bridge, alias1.vnic, dbInsVnic); err == nil {
		t.Error("cn.dbUpdate failed", cnt)
	}
	if cnt, err := cn.dbUpdate(alias.bridge, "", dbInsBr); err == nil {
		t.Error("cn.dbUpdate failed", cnt)
	}
}
