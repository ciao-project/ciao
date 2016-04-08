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

// and a concentator instance CNCI.
// The CN code will be abstracted and presented as a API that can be used
// by the launcher to create a VNIC
// The CNCI code will be run within a CNCI daemon that listens to messages on
// on SNTP

package libsnnet_test

import (
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/01org/ciao/networking/libsnnet"
)

var cnciNetEnv string

func cnciinit() {
	cnciNetEnv = os.Getenv("SNNET_ENV")

	if cnNetEnv == "" {
		cnNetEnv = "10.3.66.0/24"
	}
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
	cnci := &libsnnet.Cnci{}

	cnci.NetworkConfig = &libsnnet.NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          libsnnet.GreTunnel,
	}

	cnci.ID = "cnciuuid"

	cnciinit()
	_, net1, _ := net.ParseCIDR(cnNetEnv)
	_, net2, _ := net.ParseCIDR("192.168.1.0/24")

	mgtNet := []net.IPNet{*net1, *net2}
	cnci.ManagementNet = mgtNet
	cnci.ComputeNet = mgtNet

	if err := cnci.Init(); err != nil {
		t.Fatal(err)
	}

	if err := cnci.RebuildTopology(); err != nil {
		t.Fatal(err)
	}

	if _, err := cnci.AddRemoteSubnet(*net2, 1234, net.ParseIP("192.168.0.102")); err != nil {
		t.Error(err)
	}

	if _, err := cnci.AddRemoteSubnet(*net2, 1234, net.ParseIP("192.168.0.103")); err != nil {
		t.Error(err)
	}

	if _, err := cnci.AddRemoteSubnet(*net2, 1234, net.ParseIP("192.168.0.104")); err != nil {
		t.Error(err)
	}

	if err := cnci.DelRemoteSubnet(*net2, 1234, net.ParseIP("192.168.0.102")); err != nil {
		t.Error(err)
	}

	if err := cnci.RebuildTopology(); err != nil {
		t.Fatal(err)
	}

	if _, err := cnci.AddRemoteSubnet(*net2, 1234, net.ParseIP("192.168.0.105")); err != nil {
		t.Error(err)
	}

	if err := cnci.DelRemoteSubnet(*net2, 1234, net.ParseIP("192.168.0.103")); err != nil {
		t.Error(err)
	}

	if err := cnci.DelRemoteSubnet(*net2, 1234, net.ParseIP("192.168.0.105")); err != nil {
		t.Error(err)
	}

	if err := cnci.DelRemoteSubnet(*net2, 1234, net.ParseIP("192.168.0.102")); err != nil {
		t.Error(err)
	}
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
//depdencies and catch any dependency errors
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
	bridge, _ := libsnnet.NewBridge(bridgeAlias)

	if err := bridge.Create(); err != nil {
		t.Errorf("Bridge creation failed: %v", err)
	}
	defer bridge.Destroy()

	if err := bridge.Enable(); err != nil {
		t.Errorf("Bridge enable failed: %v", err)
	}

	// Attach the DNS masq against the CNCI bridge. This gives it an IP address
	d, err := libsnnet.NewDnsmasq(bridgeAlias, tenantUUID, subnet, reserved, bridge)

	if err != nil {
		t.Errorf("DNS Masq New failed: %v", err)
	}

	if err := d.Start(); err != nil {
		t.Errorf("DNS Masq Start: %v", err)
	}
	defer d.Stop()

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

	gre, _ := libsnnet.NewGreTunEP(greAlias, local, remote, key)

	if err := gre.Create(); err != nil {
		t.Errorf("GRE Tunnel Creation failed: %v", err)
	}
	defer gre.Destroy()

	if err := gre.Attach(bridge); err != nil {
		t.Errorf("GRE Tunnel attach failed: %v", err)
	}

	if err := gre.Enable(); err != nil {
		t.Errorf("GRE Tunnel enable failed: %v", err)
	}
}
