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
	"fmt"
	"net"
	"testing"
)

func cnciTestInit() (*Cnci, error) {
	snTestInit()

	_, testNet, err := net.ParseCIDR(snTestNet)
	if err != nil {
		return nil, err
	}

	netConfig := &NetworkConfig{
		ManagementNet: []net.IPNet{*testNet},
		ComputeNet:    []net.IPNet{*testNet},
		Mode:          GreTunnel,
	}

	cnci := &Cnci{
		ID:            "TestCNUUID",
		NetworkConfig: netConfig,
	}

	if err := cnci.Init(); err != nil {
		return nil, err
	}
	if err := cnci.RebuildTopology(); err != nil {
		return nil, err
	}

	return cnci, nil

}

//Tests all CNCI APIs
//
//Tests all operations typically performed on a CNCI
//Test includes adding and deleting a remote subnet
//Rebuild of the topology database (to simulate agent crash)
//It also tests the reset of the node to clean status
//
//Test should pass ok
func TestCNCI_Init(t *testing.T) {

	cnci, err := cnciTestInit()
	if err != nil {
		t.Fatal(err)
	}

	_, tnet, _ := net.ParseCIDR("192.168.0.0/24")

	if _, err := cnci.AddRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.102")); err != nil {
		t.Error(err)
	}
	//Duplicate
	if _, err := cnci.AddRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.102")); err != nil {
		t.Error(err)
	}

	if _, err := cnci.AddRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.103")); err != nil {
		t.Error(err)
	}

	if _, err := cnci.AddRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.104")); err != nil {
		t.Error(err)
	}

	if err := cnci.DelRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.102")); err != nil {
		t.Error(err)
	}

	if err := cnci.RebuildTopology(); err != nil {
		t.Fatal(err)
	}
	//Duplicate
	if err := cnci.RebuildTopology(); err != nil {
		t.Fatal(err)
	}

	if _, err := cnci.AddRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.105")); err != nil {
		t.Error(err)
	}

	if err := cnci.DelRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.103")); err != nil {
		t.Error(err)
	}

	if err := cnci.DelRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.105")); err != nil {
		t.Error(err)
	}
	//Duplicate
	if err := cnci.DelRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.105")); err != nil {
		t.Error(err)
	}

	if err := cnci.DelRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.102")); err != nil {
		t.Error(err)
	}
	if err := cnci.Shutdown(); err != nil {
		t.Fatal(err)
	}
	//Duplicate
	if err := cnci.Shutdown(); err != nil {
		t.Fatal(err)
	}
}

//Whitebox test case of CNCI API primitives
//
//This tests ensure that the lower level primitive
//APIs that the CNCI uses are still sane and function
//as expected. This test is expected to catch any
//issues due to change in the underlying libraries
//kernel features and applications (like dnsmasq,
//netlink) that the CNCI API relies on
//The goal of this test is to ensure we can rebase our
//dependencies and catch any dependency errors
//
//Test is expected to pass
func TestCNCI_Internal(t *testing.T) {

	// Typical inputs in YAML
	tenantUUID := "tenantUuid"
	concUUID := "concUuid"
	cnUUID := "cnUuid"
	subnetUUID := "subnetUuid"
	subnetKey := uint32(0xF)
	reserved := 10
	cnciIP := net.IPv4(127, 0, 0, 1)
	subnet := net.IPNet{
		IP:   net.IPv4(192, 168, 1, 0),
		Mask: net.IPv4Mask(255, 255, 255, 0),
	}
	// +The DHCP configuration, MAC to IP mapping is another inputs
	// This will be sent a-priori or based on design each time an instance is created

	// Create the CNCI aggregation bridge
	bridgeAlias := fmt.Sprintf("br_%s_%s_%s", tenantUUID, subnetUUID, concUUID)
	bridge, _ := newBridge(bridgeAlias)

	if err := bridge.create(); err != nil {
		t.Errorf("Bridge creation failed: %v", err)
	}
	defer func() { _ = bridge.destroy() }()

	if err := bridge.enable(); err != nil {
		t.Errorf("Bridge enable failed: %v", err)
	}

	// Attach the DNS masq against the CNCI bridge. This gives it an IP address
	d, err := newDnsmasq(bridgeAlias, tenantUUID, subnet, reserved, bridge)

	if err != nil {
		t.Errorf("DNS Masq New failed: %v", err)
	}

	if err := d.start(); err != nil {
		t.Errorf("DNS Masq Start: %v", err)
	}
	defer func() { _ = d.stop() }()

	// At this time the bridge is ready waiting for tunnels to be created
	// The next step will happen each time a tenant bridge is created for
	// this tenant on a CN
	cnIP := net.IPv4(127, 0, 0, 1)

	// Wait on SNTP messages requesting tunnel creation
	// Create a GRE tunnel that connects a tenant bridge to the CNCI bridge
	// for that subnet. The CNCI will have many bridges one for each subnet
	// the belongs to the tenant
	greAlias := fmt.Sprintf("gre_%s_%s_%s", tenantUUID, subnetUUID, cnUUID)
	local := cnciIP
	remote := cnIP
	key := subnetKey

	gre, _ := newGreTunEP(greAlias, local, remote, key)

	if err := gre.create(); err != nil {
		t.Errorf("GRE Tunnel Creation failed: %v", err)
	}
	defer func() { _ = gre.destroy() }()

	if err := gre.attach(bridge); err != nil {
		t.Errorf("GRE Tunnel attach failed: %v", err)
	}

	if err := gre.enable(); err != nil {
		t.Errorf("GRE Tunnel enable failed: %v", err)
	}
}
