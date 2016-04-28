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
	"net"
	"os"
	"strconv"
	"testing"

	"github.com/01org/ciao/networking/libsnnet"
)

var cnNetEnv string

func cninit() {
	cnNetEnv = os.Getenv("SNNET_ENV")

	if cnNetEnv == "" {
		cnNetEnv = "192.168.0.0/24"
	}
}

//Tests the scaling of the CN VNIC Creation
//
//This tests creates a large number of VNICs across a number
//of subnets
//
//Test should pass OK
func TestCN_Scaling(t *testing.T) {
	cn := &libsnnet.ComputeNode{}

	cn.NetworkConfig = &libsnnet.NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          libsnnet.GreTunnel,
	}

	cn.ID = "cnuuid"

	cninit()
	_, mnet, _ := net.ParseCIDR(cnNetEnv)

	//From YAML, on agent init
	mgtNet := []net.IPNet{*mnet}
	cn.ManagementNet = mgtNet
	cn.ComputeNet = mgtNet

	if err := cn.Init(); err != nil {
		t.Fatal("ERROR: cn.Init failed", err)
	}
	if err := cn.ResetNetwork(); err != nil {
		t.Error("ERROR: cn.ResetNetwork failed", err)
	}
	if err := cn.DbRebuild(nil); err != nil {
		t.Fatal("ERROR: cn.dbRebuild failed")
	}

	//From YAML on instance init
	tenantID := "tenantuuid"
	concIP := net.IPv4(192, 168, 111, 1)

	var maxBridges, maxVnics int
	if testing.Short() {
		maxBridges = scaleCfg.maxBridgesShort
		maxVnics = scaleCfg.maxVnicsShort
	} else {
		maxBridges = scaleCfg.maxBridgesLong
		maxVnics = scaleCfg.maxVnicsLong
	}

	for s3 := 1; s3 <= maxBridges; s3++ {
		s4 := 0
		_, tenantNet, _ := net.ParseCIDR("193.168." + strconv.Itoa(s3) + "." + strconv.Itoa(s4) + "/24")
		subnetID := "suuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)

		for s4 := 2; s4 <= maxVnics; s4++ {

			vnicIP := net.IPv4(192, 168, byte(s3), byte(s4))
			mac, _ := net.ParseMAC("CA:FE:00:01:02:03")

			vnicID := "vuuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)
			instanceID := "iuuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)
			vnicCfg := &libsnnet.VnicConfig{
				VnicIP:     vnicIP,
				ConcIP:     concIP,
				VnicMAC:    mac,
				Subnet:     *tenantNet,
				SubnetKey:  s3,
				VnicID:     vnicID,
				InstanceID: instanceID,
				SubnetID:   subnetID,
				TenantID:   tenantID,
				ConcID:     "cnciuuid",
			}

			if vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg); err != nil {
				t.Error("ERROR: cn.CreateVnic  failed", err)
			} else {
				t.Logf("VNIC Created vnic[%v] cfg[%v] event[%v]", vnic, vnicCfg, ssntpEvent)
			}
		}
	}

	for s3 := 1; s3 <= maxBridges; s3++ {
		s4 := 0
		_, tenantNet, _ := net.ParseCIDR("193.168." + strconv.Itoa(s3) + "." + strconv.Itoa(s4) + "/24")
		subnetID := "suuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)

		for s4 := 2; s4 <= maxVnics; s4++ {

			vnicIP := net.IPv4(192, 168, byte(s3), byte(s4))
			mac, _ := net.ParseMAC("CA:FE:00:01:02:03")

			vnicID := "vuuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)
			instanceID := "iuuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)
			vnicCfg := &libsnnet.VnicConfig{
				VnicIP:     vnicIP,
				ConcIP:     concIP,
				VnicMAC:    mac,
				Subnet:     *tenantNet,
				SubnetKey:  0xF,
				VnicID:     vnicID,
				InstanceID: instanceID,
				SubnetID:   subnetID,
				TenantID:   tenantID,
				ConcID:     "cnciuuid",
			}

			if ssntpEvent, _, err := cn.DestroyVnic(vnicCfg); err != nil {
				t.Error("ERROR: cn.DestroyVnic failed event", vnicCfg, err)
			} else {
				t.Logf("VNIC Destroyed cfg[%v] event[%v]", vnicCfg, ssntpEvent)
			}
		}
	}
}

//Tests the ResetNetwork API
//
//This test creates multiple VNICs belonging to multiple tenants
//It then uses the ResetNetwork API to reset the node's networking
//state to a clean state (as in reset). This test also check that
//the API can be called midway through a node's lifecycle and
//the DbRebuild API can be used to re-construct the node's
//networking state
//
//Test should pass OK
func TestCN_ResetNetwork(t *testing.T) {
	cn := &libsnnet.ComputeNode{}

	cn.NetworkConfig = &libsnnet.NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          libsnnet.GreTunnel,
	}

	cn.ID = "cnuuid"

	_, net2, _ := net.ParseCIDR("193.168.1.0/24")
	cninit()
	_, net3, _ := net.ParseCIDR(cnNetEnv)

	//From YAML, on agent init
	mgtNet := []net.IPNet{*net2, *net3}
	cn.ManagementNet = mgtNet
	cn.ComputeNet = mgtNet

	if err := cn.Init(); err != nil {
		t.Fatal("ERROR: cn.Init failed", err)
	}
	if err := cn.ResetNetwork(); err != nil {
		t.Error("ERROR: cn.ResetNetwork failed", err)
	}

	//From YAML on instance init
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	vnicCfg := &libsnnet.VnicConfig{
		VnicIP:     net.IPv4(192, 168, 1, 100),
		ConcIP:     net.IPv4(192, 168, 1, 1),
		VnicMAC:    mac,
		Subnet:     *net2,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	if vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg); err != nil {
		t.Error("ERROR: cn.CreateVnic  failed", err)
	} else {
		if ssntpEvent == nil {
			t.Error("ERROR: cn.CreateVnic expected event", vnic)
		}
		t.Log("VNIC1 duplicate created =", vnic.LinkName, ssntpEvent)
	}

	vnicCfg.TenantID = "tuuid2"
	vnicCfg.ConcIP = net.IPv4(192, 168, 1, 2)

	if vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg); err != nil {
		t.Error("ERROR: cn.CreateVnic  failed", err)
	} else {
		if ssntpEvent == nil {
			t.Error("ERROR: cn.CreateVnic expected event", vnic)
		}
		t.Log("VNIC1 duplicate created =", vnic.LinkName, ssntpEvent)
	}

	if err := cn.ResetNetwork(); err != nil {
		t.Error("ERROR: cn.ResetNetwork failed", err)
	}
	if err := cn.DbRebuild(nil); err != nil {
		t.Fatal("ERROR: cn.dbRebuild failed")
	}

	vnicCfg.TenantID = "tuuid"
	vnicCfg.ConcIP = net.IPv4(192, 168, 1, 1)

	if vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg); err != nil {
		t.Error("ERROR: cn.CreateVnic  failed", err)
	} else {
		if ssntpEvent == nil {
			t.Error("ERROR: cn.CreateVnic expected event", vnic)
		}
		t.Log("VNIC1 duplicate created =", vnic.LinkName, ssntpEvent)
	}

	vnicCfg.TenantID = "tuuid2"
	vnicCfg.ConcIP = net.IPv4(192, 168, 1, 2)

	if vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg); err != nil {
		t.Error("ERROR: cn.CreateVnic  failed", err)
	} else {
		if ssntpEvent == nil {
			t.Error("ERROR: cn.CreateVnic expected event", vnic)
		}
		t.Log("VNIC1 duplicate created =", vnic.LinkName, ssntpEvent)
	}

	if err := cn.ResetNetwork(); err != nil {
		t.Error("ERROR: cn.ResetNetwork failed", err)
	}

}

//Tests multiple VNIC's creation
//
//This tests tests if multiple VNICs belonging to multiple
//tenants can be successfully created and deleted on a given CN
//This tests also checks for the generation of the requisite
//SSNTP message that the launcher is expected to send to the
//CNCI via the scheduler
//
//Test should pass OK
func TestCN_MultiTenant(t *testing.T) {
	cn := &libsnnet.ComputeNode{}

	cn.NetworkConfig = &libsnnet.NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          libsnnet.GreTunnel,
	}

	cn.ID = "cnuuid"

	//_, net1, _ := net.ParseCIDR("127.0.0.0/24")
	_, net2, _ := net.ParseCIDR("193.168.1.0/24")
	cninit()
	_, net3, _ := net.ParseCIDR(cnNetEnv)

	//From YAML, on agent init
	mgtNet := []net.IPNet{*net2, *net3}
	cn.ManagementNet = mgtNet
	cn.ComputeNet = mgtNet

	if err := cn.Init(); err != nil {
		t.Fatal("ERROR: cn.Init failed", err)
	}
	if err := cn.ResetNetwork(); err != nil {
		t.Error("ERROR: cn.ResetNetwork failed", err)
	}
	if err := cn.DbRebuild(nil); err != nil {
		t.Fatal("ERROR: cn.dbRebuild failed")
	}
	if err := cn.ResetNetwork(); err != nil {
		t.Error("ERROR: cn.ResetNetwork failed", err)
	}
	if err := cn.DbRebuild(nil); err != nil {
		t.Fatal("ERROR: cn.dbRebuild failed")
	}

	//From YAML on instance init
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	vnicCfg := &libsnnet.VnicConfig{
		VnicIP:     net.IPv4(192, 168, 1, 100),
		ConcIP:     net.IPv4(192, 168, 1, 1),
		VnicMAC:    mac,
		Subnet:     *net2,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	if vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg); err != nil {
		t.Error("ERROR: cn.CreateVnic  failed", err)
	} else {
		if ssntpEvent == nil {
			t.Error("ERROR: cn.CreateVnic expected event", vnic)
		}
		t.Log("VNIC1 duplicate created =", vnic.LinkName, ssntpEvent)
	}

	vnicCfg.TenantID = "tuuid2"
	vnicCfg.ConcIP = net.IPv4(192, 168, 1, 2)

	if vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg); err != nil {
		t.Error("ERROR: cn.CreateVnic  failed", err)
	} else {
		if ssntpEvent == nil {
			t.Error("ERROR: cn.CreateVnic expected event", vnic)
		}
		t.Log("VNIC1 duplicate created =", vnic.LinkName, ssntpEvent)
	}

	if ssntpEvent, _, err := cn.DestroyVnic(vnicCfg); err != nil {
		t.Error("ERROR: cn.DestroyVnic failed event", vnicCfg, err)
	} else {
		if ssntpEvent == nil {
			t.Error("ERROR: cn.DestroyVnic expected event", vnicCfg, err)
		}
	}

	vnicCfg.TenantID = "tuuid"
	vnicCfg.ConcIP = net.IPv4(192, 168, 1, 1)

	if ssntpEvent, _, err := cn.DestroyVnic(vnicCfg); err != nil {
		t.Error("ERROR: cn.DestroyVnic failed event", vnicCfg, err)
	} else {
		if ssntpEvent == nil {
			t.Error("ERROR: cn.DestroyVnic expected event", vnicCfg, err)
		}
	}
}

//Negative tests for CN API
//
//This tests for various invalid API invocations
//This test has to be greatly enhanced.
//
//Test is expected to pass
func TestCN_Negative(t *testing.T) {
	cn := &libsnnet.ComputeNode{}

	cn.NetworkConfig = &libsnnet.NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          libsnnet.GreTunnel,
	}

	cn.ID = "cnuuid"

	//_, net1, _ := net.ParseCIDR("127.0.0.0/24")
	_, net2, _ := net.ParseCIDR("193.168.1.0/24")
	cninit()
	_, net3, _ := net.ParseCIDR(cnNetEnv)

	//From YAML, on agent init
	mgtNet := []net.IPNet{*net2, *net3}
	cn.ManagementNet = mgtNet
	cn.ComputeNet = mgtNet

	if err := cn.Init(); err != nil {
		t.Fatal("ERROR: cn.Init failed", err)
	}

	if err := cn.DbRebuild(nil); err != nil {
		t.Fatal("ERROR: cn.dbRebuild failed")
	}

	//From YAML on instance init
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	vnicCfg := &libsnnet.VnicConfig{
		VnicIP:     net.IPv4(192, 168, 1, 100),
		ConcIP:     net.IPv4(192, 168, 1, 1),
		VnicMAC:    mac,
		Subnet:     *net2,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		//TenantID:   "tuuid", Leaving it blank should fail
		SubnetID: "suuid",
		ConcID:   "cnciuuid",
	}

	if vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg); err != nil {
		t.Log("ERROR: cn.CreateVnic should have failed", err)
	} else {
		//Launcher will attach to this name and send out the event
		t.Error("Failure expected VNIC created =", vnic.LinkName, ssntpEvent)
	}

	//Fix the errors
	vnicCfg.TenantID = "tuuid"

	// Try and create it again.
	var vnicName string
	if vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg); err != nil {
		t.Error("ERROR: cn.CreateVnic  failed", err)
	} else {
		if ssntpEvent == nil {
			t.Error("ERROR: cn.CreateVnic expected event", vnic)
		}
		t.Log("VNIC1 duplicate created =", vnic.LinkName, ssntpEvent)
		vnicName = vnic.LinkName
	}

	//Try and create a duplicate. Should work
	if vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg); err != nil {
		t.Error("ERROR: cn.CreateVnic  failed", err)
	} else {
		if ssntpEvent != nil {
			t.Error("ERROR: cn.CreateVnic unexpected event", vnic, vnicCfg, ssntpEvent)
		}
		t.Log("VNIC1 duplicate created =", vnic.LinkName, ssntpEvent)
		if vnicName != vnic.LinkName {
			t.Error("ERROR: VNIC names do not match", vnicName, vnic.LinkName)
		}
	}

	// Try and destroy
	if ssntpEvent, _, err := cn.DestroyVnic(vnicCfg); err != nil {
		t.Error("ERROR: cn.DestroyVnic failed event", vnicCfg, err)
	} else {
		if ssntpEvent == nil {
			t.Error("ERROR: cn.DestroyVnic expected event", vnicCfg, err)
		}
	}
}

//Tests a node can serve as both CN and NN simultaneously
//
//This test checks that at least from the networking point
//of view we can create both Instance VNICs and CNCI VNICs
//on the same node. Even though the launcher does not
//support this mode today, the networking layer allows
//creation and co-existence of both type of VNICs on the
//same node and they will both work
//
//Test should pass OK
func TestCN_AndNN(t *testing.T) {
	cn := &libsnnet.ComputeNode{}

	cn.NetworkConfig = &libsnnet.NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          libsnnet.GreTunnel,
	}

	cn.ID = "cnuuid"

	//_, net1, _ := net.ParseCIDR("127.0.0.0/24") //Add this so that init will pass
	_, net1, _ := net.ParseCIDR("192.168.0.0/24")
	_, net2, _ := net.ParseCIDR("192.168.1.0/24")
	cninit()
	_, net3, _ := net.ParseCIDR(cnNetEnv)

	mgtNet := []net.IPNet{*net1, *net2, *net3}
	cn.ManagementNet = mgtNet
	cn.ComputeNet = mgtNet

	if err := cn.Init(); err != nil {
		t.Fatal("ERROR: cn.Init failed", err)
	}

	if err := cn.DbRebuild(nil); err != nil {
		t.Fatal("ERROR: cn.dbRebuild failed")
	}

	//From YAML on instance init
	cnciMac, _ := net.ParseMAC("CA:FE:CC:01:02:03")
	cnciVnicCfg := &libsnnet.VnicConfig{
		VnicRole:   libsnnet.DataCenter,
		VnicMAC:    cnciMac,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
	}

	// Create a VNIC
	var cnciVnic1Name string
	if cnciVnic, err := cn.CreateCnciVnic(cnciVnicCfg); err != nil {
		t.Error("ERROR: cn.CreateCnciVnic failed", err)
	} else {
		//Launcher will attach to this name and send out the event
		t.Log("VNIC1 created =", cnciVnic.LinkName)
		cnciVnic1Name = cnciVnic.LinkName
	}

	var cnciVnic1DupName string
	// Try and create it again. Should return cached value
	if cnciVnic, err := cn.CreateCnciVnic(cnciVnicCfg); err != nil {
		t.Error("ERROR: cn.CreateVnic duplicate failed", err)
	} else {
		t.Log("VNIC1 duplicate created =", cnciVnic.LinkName)
		cnciVnic1DupName = cnciVnic.LinkName
	}

	if cnciVnic1Name != cnciVnic1DupName {
		t.Error("ERROR: cn.CreateCnciVnic VNIC1 and VNIC1 Dup interface names do not match", cnciVnic1Name, cnciVnic1DupName)
	}
	t.Log("cn.CreateVnic VNIC1 and VNIC1 Dup interface names", cnciVnic1Name, cnciVnic1DupName)

	//From YAML on instance init
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	vnicCfg := &libsnnet.VnicConfig{
		VnicIP:     net.IPv4(192, 168, 1, 100),
		ConcIP:     net.IPv4(192, 168, 1, 1),
		VnicMAC:    mac,
		Subnet:     *net2,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	// Create a VNIC: Should create bridge and tunnels
	var vnic1Name, vnic1DupName string
	if vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg); err != nil {
		t.Error("ERROR: cn.CreateVnic failed", err)
	} else {
		//We expect a bridge creation event
		if ssntpEvent == nil {
			t.Error("ERROR: cn.CreateVnic expected event", vnic, ssntpEvent)
		}
		//Launcher will attach to this name and send out the event
		t.Log("VNIC1 created =", vnic.LinkName, ssntpEvent)
		vnic1Name = vnic.LinkName
	}

	// Try and create it again. Should return cached value
	if vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg); err != nil {
		t.Error("ERROR: cn.CreateVnic duplicate failed", err, ssntpEvent)
	} else {
		//We do not expect a bridge creation event
		if ssntpEvent != nil {
			t.Error("ERROR: cn.CreateVnic duplicate unexpected event", vnic, ssntpEvent)
		}
		//Launcher will attach to this name and send out the event
		t.Log("VNIC1 duplicate created =", vnic.LinkName, ssntpEvent)
		vnic1DupName = vnic.LinkName
	}

	if vnic1Name != vnic1DupName {
		t.Error("ERROR: cn.CreateVnic VNIC1 and VNIC2 interface names do not match", vnic1Name, vnic1DupName)
	}
	t.Log("cn.CreateVnic VNIC1 and VNIC2 interface names", vnic1Name, vnic1DupName)

	mac2, _ := net.ParseMAC("CA:FE:00:01:02:22")
	vnicCfg2 := &libsnnet.VnicConfig{
		VnicIP:     net.IPv4(192, 168, 1, 2),
		ConcIP:     net.IPv4(192, 168, 1, 1),
		VnicMAC:    mac2,
		Subnet:     *net2,
		SubnetKey:  0xF,
		VnicID:     "vuuid2",
		InstanceID: "iuuid2",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	// Create a second VNIC on the same tenant subnet
	if vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg2); err != nil {
		t.Error("ERROR: cn.CreateVnic VNIC2 failed", err, ssntpEvent)
	} else {
		//We do not expect a bridge creation event
		if ssntpEvent != nil {
			t.Error("ERROR: cn.CreateVnic VNIC2 unexpected event", vnic, ssntpEvent)
		}
		//Launcher will attach to this name and send out the event
		t.Log("VNIC2 created =", vnic.LinkName, ssntpEvent)
	}

	if ssntpEvent, _, err := cn.DestroyVnic(vnicCfg2); err != nil {
		if ssntpEvent != nil {
			t.Error("ERROR: cn.DestroyVnic VNIC2 unexpected event", err, ssntpEvent)
		}
		t.Error("ERROR: cn.DestroyVnic VNIC2 destroy attempt failed", err)
	}

	cnciMac2, _ := net.ParseMAC("CA:FE:CC:01:02:22")
	cnciVnicCfg2 := &libsnnet.VnicConfig{
		VnicRole:   libsnnet.DataCenter,
		VnicMAC:    cnciMac2,
		VnicID:     "vuuid2",
		InstanceID: "iuuid2",
		TenantID:   "tuuid2",
	}

	// Create and destroy a second VNIC
	if cnciVnic, err := cn.CreateCnciVnic(cnciVnicCfg2); err != nil {
		t.Error("ERROR: cn.CreateVnic VNIC2 failed", err)
	} else {
		//Launcher will attach to this name
		t.Log("VNIC2 created =", cnciVnic.LinkName)
	}
	if err := cn.DestroyCnciVnic(cnciVnicCfg2); err != nil {
		t.Error("ERROR: cn.DestroyCnciVnic VNIC2 destroy attempt failed", err)
	}

	// Destroy the first VNIC
	if err := cn.DestroyCnciVnic(cnciVnicCfg); err != nil {
		t.Error("ERROR: cn.DestroyCnciVnic VNIC1 failed", err)
	}

	// Try and destroy it again - should work
	if err := cn.DestroyCnciVnic(cnciVnicCfg); err != nil {
		t.Error("ERROR: cn.DestroyCnciVnic VNIC1 duplicate destroy attempt failed", err)
	}

	// Destroy the first VNIC - Deletes the bridge and tunnel
	if ssntpEvent, _, err := cn.DestroyVnic(vnicCfg); err != nil {
		t.Error("ERROR: cn.DestroyVnic VNIC1 failed", err)
	} else {
		//We expect a bridge deletion event
		if ssntpEvent == nil {
			t.Error("ERROR: cn.DestroyVnic VNIC1 expected event")
		}
		//Launcher will send this event out
		t.Log("cn.Destroy VNIC1 ssntp event", ssntpEvent)
	}

	// Try and destroy it again - should work
	if ssntpEvent, _, err := cn.DestroyVnic(vnicCfg); err != nil {
		if ssntpEvent != nil {
			t.Error("ERROR: cn.DestroyVnic VNIC1 duplicate unexpected event", err, ssntpEvent)
		}
		t.Error("ERROR: cn.DestroyVnic VNIC1 duplicate destroy attempt failed", err)
	}
}

//Tests typical sequence of NN APIs
//
//This tests exercises the standard set of APIs that
//the launcher invokes when setting up a NN and creating
//VNICs. It check for duplicate VNIC creation, duplicate
//VNIC deletion
//
//Test is expected to pass
func TestNN_Base(t *testing.T) {
	cn := &libsnnet.ComputeNode{}

	cn.NetworkConfig = &libsnnet.NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          libsnnet.GreTunnel,
	}

	cn.ID = "cnuuid"

	//_, net1, _ := net.ParseCIDR("127.0.0.0/24") //Add this so that init will pass
	_, net1, _ := net.ParseCIDR("193.168.0.0/24")
	_, net2, _ := net.ParseCIDR("193.168.1.0/24")
	cninit()
	_, net3, _ := net.ParseCIDR(cnNetEnv)

	mgtNet := []net.IPNet{*net1, *net2, *net3}
	cn.ManagementNet = mgtNet
	cn.ComputeNet = mgtNet

	if err := cn.Init(); err != nil {
		t.Fatal("ERROR: cn.Init failed", err)
	}

	if err := cn.DbRebuild(nil); err != nil {
		t.Fatal("ERROR: cn.dbRebuild failed")
	}

	//From YAML on instance init
	cnciMac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	cnciVnicCfg := &libsnnet.VnicConfig{
		VnicRole:   libsnnet.DataCenter,
		VnicMAC:    cnciMac,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
	}

	// Create a VNIC
	var cnciVnic1Name string
	if cnciVnic, err := cn.CreateCnciVnic(cnciVnicCfg); err != nil {
		t.Error("ERROR: cn.CreateCnciVnic failed", err)
	} else {
		//Launcher will attach to this name and send out the event
		t.Log("VNIC1 created =", cnciVnic.LinkName)
		cnciVnic1Name = cnciVnic.LinkName
	}

	var cnciVnic1DupName string
	// Try and create it again. Should return cached value
	if cnciVnic, err := cn.CreateCnciVnic(cnciVnicCfg); err != nil {
		t.Error("ERROR: cn.CreateVnic duplicate failed", err)
	} else {
		t.Log("VNIC1 duplicate created =", cnciVnic.LinkName)
		cnciVnic1DupName = cnciVnic.LinkName
	}

	if cnciVnic1Name != cnciVnic1DupName {
		t.Error("ERROR: cn.CreateCnciVnic VNIC1 and VNIC1 Dup interface names do not match", cnciVnic1Name, cnciVnic1DupName)
	}
	t.Log("cn.CreateVnic VNIC1 and VNIC1 Dup interface names", cnciVnic1Name, cnciVnic1DupName)

	cnciMac2, _ := net.ParseMAC("CA:FE:00:01:02:22")
	cnciVnicCfg2 := &libsnnet.VnicConfig{
		VnicRole:   libsnnet.DataCenter,
		VnicMAC:    cnciMac2,
		VnicID:     "vuuid2",
		InstanceID: "iuuid2",
		TenantID:   "tuuid2",
	}

	// Create and destroy a second VNIC
	if cnciVnic, err := cn.CreateCnciVnic(cnciVnicCfg2); err != nil {
		t.Error("ERROR: cn.CreateVnic VNIC2 failed", err)
	} else {
		//Launcher will attach to this name
		t.Log("VNIC2 created =", cnciVnic.LinkName)
	}
	if err := cn.DestroyCnciVnic(cnciVnicCfg2); err != nil {
		t.Error("ERROR: cn.DestroyCnciVnic VNIC2 destroy attempt failed", err)
	}

	// Destroy the first VNIC
	if err := cn.DestroyCnciVnic(cnciVnicCfg); err != nil {
		t.Error("ERROR: cn.DestroyCnciVnic VNIC1 failed", err)
	}

	// Try and destroy it again - should work
	if err := cn.DestroyCnciVnic(cnciVnicCfg); err != nil {
		t.Error("ERROR: cn.DestroyCnciVnic VNIC1 duplicate destroy attempt failed", err)
	}
}

func validSsntpEvent(ssntpEvent *libsnnet.SsntpEventInfo, cfg *libsnnet.VnicConfig) error {

	if ssntpEvent == nil {
		return fmt.Errorf("SsntpEvent: nil")
	}

	//Note: Checking for non nil values just to ensure the caller called it with all
	//parameters setup properly.
	switch {
	case ssntpEvent.ConcID != cfg.ConcID:
	case ssntpEvent.ConcID == "":

	case ssntpEvent.CnciIP != cfg.ConcIP.String():
	case ssntpEvent.CnciIP == "":

	//case ssntpEvent.CnIP != has to be set by the caller

	case ssntpEvent.Subnet != cfg.Subnet.String():
	case ssntpEvent.Subnet == "":

	case ssntpEvent.SubnetKey != cfg.SubnetKey:
	case ssntpEvent.SubnetKey == 0:
	case ssntpEvent.SubnetKey == -1:

	case ssntpEvent.SubnetID != cfg.SubnetID:
	case ssntpEvent.SubnetID == "":

	case ssntpEvent.TenantID != cfg.TenantID:
	case ssntpEvent.TenantID == "":
	default:
		return nil
	}

	return fmt.Errorf("SsntpEvent: fields do not match %v != %v", ssntpEvent, cfg)
}

//Tests typical sequence of CN APIs
//
//This tests exercises the standard set of APIs that
//the launcher invokes when setting up a CN and creating
//VNICs. It check for duplicate VNIC creation, duplicate
//VNIC deletion
//
//Test is expected to pass
func TestCN_Base(t *testing.T) {
	cn := &libsnnet.ComputeNode{}

	cn.NetworkConfig = &libsnnet.NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          libsnnet.GreTunnel,
	}

	cn.ID = "cnuuid"

	_, net1, _ := net.ParseCIDR("127.0.0.0/24") //Add this so that init will pass
	_, net2, _ := net.ParseCIDR("193.168.1.0/24")
	cninit()
	_, net3, _ := net.ParseCIDR(cnNetEnv)

	mgtNet := []net.IPNet{*net2}
	cn.ManagementNet = mgtNet
	cn.ComputeNet = mgtNet

	//Negative
	if err := cn.Init(); err == nil {
		t.Error("ERROR: cn.Init failed", err)
	}

	//From YAML, on agent init
	mgtNet = []net.IPNet{*net1, *net2, *net3}
	cn.ManagementNet = mgtNet
	cn.ComputeNet = mgtNet

	if err := cn.Init(); err != nil {
		t.Fatal("ERROR: cn.Init failed", err)
	}

	if err := cn.DbRebuild(nil); err != nil {
		t.Fatal("ERROR: cn.dbRebuild failed")
	}

	//From YAML on instance init
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	vnicCfg := &libsnnet.VnicConfig{
		VnicIP:     net.IPv4(192, 168, 1, 100),
		ConcIP:     net.IPv4(192, 168, 1, 1),
		VnicMAC:    mac,
		Subnet:     *net2,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	// Create a VNIC: Should create bridge and tunnels
	var vnic1Name, vnic1DupName string
	if vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg); err != nil {
		t.Error("ERROR: cn.CreateVnic failed", err)
	} else {
		//We expect a bridge creation event
		if ssntpEvent == nil {
			t.Error("ERROR: cn.CreateVnic expected event", vnic, ssntpEvent)
		}
		if ssntpEvent != nil {
			//Check the fields of the ssntpEvent
			if err := validSsntpEvent(ssntpEvent, vnicCfg); err != nil {
				t.Errorf("ERROR: cn.CreateVnic event population errror %v ", err)
			}
			if ssntpEvent.Event != libsnnet.SsntpTunAdd {
				t.Error("ERROR: cn.CreateVnic event population errror", vnic, ssntpEvent)
			}
		}
		//Launcher will attach to this name and send out the event
		t.Log("VNIC1 created =", vnic.LinkName, ssntpEvent)
		vnic1Name = vnic.LinkName
	}

	// Try and create it again. Should return cached value
	if vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg); err != nil {
		t.Error("ERROR: cn.CreateVnic duplicate failed", err, ssntpEvent)
	} else {
		//We do not expect a bridge creation event
		if ssntpEvent != nil {
			t.Error("ERROR: cn.CreateVnic duplicate unexpected event", vnic, ssntpEvent)
		}
		//Launcher will attach to this name and send out the event
		t.Log("VNIC1 duplicate created =", vnic.LinkName, ssntpEvent)
		vnic1DupName = vnic.LinkName
	}

	if vnic1Name != vnic1DupName {
		t.Error("ERROR: cn.CreateVnic VNIC1 and VNIC2 interface names do not match", vnic1Name, vnic1DupName)
	}
	t.Log("cn.CreateVnic VNIC1 and VNIC2 interface names", vnic1Name, vnic1DupName)

	mac2, _ := net.ParseMAC("CA:FE:00:01:02:22")
	vnicCfg2 := &libsnnet.VnicConfig{
		VnicIP:     net.IPv4(192, 168, 1, 2),
		ConcIP:     net.IPv4(192, 168, 1, 1),
		VnicMAC:    mac2,
		Subnet:     *net2,
		SubnetKey:  0xF,
		VnicID:     "vuuid2",
		InstanceID: "iuuid2",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	// Create a second VNIC on the same tenant subnet
	if vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg2); err != nil {
		t.Error("ERROR: cn.CreateVnic VNIC2 failed", err, ssntpEvent)
	} else {
		//We do not expect a bridge creation event
		if ssntpEvent != nil {
			t.Error("ERROR: cn.CreateVnic VNIC2 unexpected event", vnic, ssntpEvent)
		}
		//Launcher will attach to this name and send out the event
		t.Log("VNIC2 created =", vnic.LinkName, ssntpEvent)
	}

	if ssntpEvent, _, err := cn.DestroyVnic(vnicCfg2); err != nil {
		if ssntpEvent != nil {
			t.Error("ERROR: cn.DestroyVnic VNIC2 unexpected event", err, ssntpEvent)
		}
		t.Error("ERROR: cn.DestroyVnic VNIC2 destroy attempt failed", err)
	}

	// Destroy the first VNIC - Deletes the bridge and tunnel
	if ssntpEvent, _, err := cn.DestroyVnic(vnicCfg); err != nil {
		t.Error("ERROR: cn.DestroyVnic VNIC1 failed", err)
	} else {
		//We expect a bridge deletion event
		if ssntpEvent == nil {
			t.Error("ERROR: cn.DestroyVnic VNIC1 expected event")
		}
		if ssntpEvent != nil {
			//Check the fields of the ssntpEvent
			if err := validSsntpEvent(ssntpEvent, vnicCfg); err != nil {
				t.Errorf("ERROR: cn.DestroyVnic event population errror %v", err)
			}
			if ssntpEvent.Event != libsnnet.SsntpTunDel {
				t.Error("ERROR: cn.DestroyVnic event population errror", vnicCfg, ssntpEvent)
			}
		}
		//Launcher will send this event out
		t.Log("cn.Destroy VNIC1 ssntp event", ssntpEvent)
	}

	// Try and destroy it again - should work
	if ssntpEvent, _, err := cn.DestroyVnic(vnicCfg); err != nil {
		if ssntpEvent != nil {
			t.Error("ERROR: cn.DestroyVnic VNIC1 duplicate unexpected event", err, ssntpEvent)
		}
		t.Error("ERROR: cn.DestroyVnic VNIC1 duplicate destroy attempt failed", err)
	}
}

//Whitebox test the CN API
//
//This tests exercises tests the primitive operations
//that the CN API rely on. This is used to check any
//issues with the underlying netlink library or kernel
//This tests fails typically if the kernel or netlink
//implementation changes
//
//Test is expected to pass
func TestCN_Whitebox(t *testing.T) {
	var instanceMAC net.HardwareAddr
	var err error

	// Typical inputs in YAML from Controller
	tenantUUID := "tenantUuid"
	instanceUUID := "tenantUuid"
	subnetUUID := "subnetUuid"
	subnetKey := uint32(0xF)
	concUUID := "concUuid"
	//The IP corresponding to CNCI, maybe we can use DNS resolution here?
	concIP := net.IPv4(192, 168, 1, 1)
	//The IP corresponding to the VNIC that carries tenant traffic
	cnIP := net.IPv4(127, 0, 0, 1)
	if instanceMAC, err = net.ParseMAC("CA:FE:00:01:02:03"); err != nil {
		t.Errorf("Invalid MAC address")
	}

	// Create the CN tenant bridge only if it does not exist
	bridgeAlias := fmt.Sprintf("br_%s_%s_%s", tenantUUID, subnetUUID, concUUID)
	bridge, _ := libsnnet.NewBridge(bridgeAlias)

	if err := bridge.GetDevice(); err != nil {
		// First instance to land, create the bridge and tunnel
		if err := bridge.Create(); err != nil {
			t.Errorf("Bridge creation failed: %v", err)
		}
		defer bridge.Destroy()

		// Create the tunnel to connect to the CNCI
		local := cnIP
		remote := concIP

		greAlias := fmt.Sprintf("gre_%s_%s_%s", tenantUUID, subnetUUID, concUUID)
		gre, _ := libsnnet.NewGreTunEP(greAlias, local, remote, subnetKey)

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

		if err := bridge.Enable(); err != nil {
			t.Errorf("Bridge enable failed: %v", err)
		}

	}

	// Create the VNIC for the instance
	vnicAlias := fmt.Sprintf("vnic_%s_%s_%s_%s", tenantUUID, instanceUUID, instanceMAC, concUUID)
	vnic, _ := libsnnet.NewVnic(vnicAlias)
	vnic.MACAddr = &instanceMAC

	if err := vnic.Create(); err != nil {
		t.Errorf("Vnic Create failed: %v", err)
	}
	defer vnic.Destroy()

	if err := vnic.Attach(bridge); err != nil {
		t.Errorf("Vnic attach failed: %v", err)
	}

	if err := vnic.Enable(); err != nil {
		t.Errorf("Vnic enable failed: %v", err)
	}

	if err := bridge.Enable(); err != nil {
		t.Errorf("Bridge enable: %v", err)
	}
}
