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

//Benchmarks Worst case latency of VNIC creation
//
//BenchmarkComputeNodeWorstCase measures the time is takes
//to instantiate a VNIC on a node that does not have that
//tenant subnet present
//This means that there will be bridge created, a GRE tunnel
//created and a tap inteface created and they are all linked
//to one another. Additionally a SSNTP event is also generated
//Based on current observation most of the time is spent in the
//kernel processing the netlink message
//To ensure that we do not pollute the test system we delete
//the VNIC.
//Hence the benchmarked time includes the time it takes to
//create and delete the VNIC (not just create).
//However the deletes are more efficient than creates
//This does not truly measure the cost of synchrnoization
//when multiple launcher threads are creating VNIC simulatenously.
//However based on current measurements the cost of a channel based
//sync is about 10ms (for a uncontended channel). The mutex is almost
//free when un-contended
//
//Test should pass ok
func BenchmarkComputeNodeWorstCase(b *testing.B) {
	cn := &libsnnet.ComputeNode{}

	cn.NetworkConfig = &libsnnet.NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          libsnnet.GreTunnel,
	}

	cn.ID = "cnuuid"

	_, net1, _ := net.ParseCIDR("127.0.0.0/24") //Add this so that init will pass
	_, net2, _ := net.ParseCIDR("193.168.1.0/24")
	_, net3, _ := net.ParseCIDR("10.3.66.0/24")

	//From YAML, on agent init
	mgtNet := []net.IPNet{*net1, *net2, *net3}
	cn.ManagementNet = mgtNet
	cn.ComputeNet = mgtNet

	if err := cn.Init(); err != nil {
		b.Fatal("cn.Init failed", err)
	}

	/* Pollutes the benchmark */
	/*
		if err := cn.DbRebuild(nil); err != nil {
			b.Fatal("cn.dbRebuild failed")
		}
	*/

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

	for i := 0; i < b.N; i++ {

		if vnic, ssntpEvent, err := cn.CreateVnic(vnicCfg); err != nil {
			b.Error("cn.CreateVnic failed", err)
		} else {
			//We expect a bridge creation event
			if ssntpEvent == nil {
				b.Error("cn.CreateVnic expected event", vnic, ssntpEvent)
			}
		}

		if ssntpEvent, err := cn.DestroyVnic(vnicCfg); err != nil {
			b.Error("cn.DestroyVnic failed", err)
		} else {
			//We expect a bridge deletion event
			if ssntpEvent == nil {
				b.Error("cn.DestroyVnic expected event")
			}
		}
	}
}

//Benchmarks best case VNIC creation latency
//
//BenchmarkComputeNodeWorstCase measures the time is takes
//to instantiate a VNIC on a node that already has that
//tenant subnet present
//Hence this is just the cost to create the tap and link
//it to the brigde.
//As mentioned before this also deletes the VNIC.
//Hence the cost includes the cost to create and delete the VNIC
//
//Test should pass OK
func BenchmarkComputeNodeBestCase(b *testing.B) {
	cn := &libsnnet.ComputeNode{}

	cn.NetworkConfig = &libsnnet.NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          libsnnet.GreTunnel,
	}

	cn.ID = "cnuuid"

	_, net1, _ := net.ParseCIDR("127.0.0.0/24") //Add this so that init will pass
	_, net2, _ := net.ParseCIDR("193.168.1.0/24")
	_, net3, _ := net.ParseCIDR("10.3.66.0/24")

	//From YAML, on agent init
	mgtNet := []net.IPNet{*net1, *net2, *net3}
	cn.ManagementNet = mgtNet
	cn.ComputeNet = mgtNet

	if err := cn.Init(); err != nil {
		b.Fatal("cn.Init failed", err)
	}

	if err := cn.DbRebuild(nil); err != nil {
		b.Fatal("cn.dbRebuild failed")
	}

	//From YAML on instance init
	macSeed, _ := net.ParseMAC("CA:FE:00:01:02:ED")
	vnicCfgSeed := &libsnnet.VnicConfig{
		VnicIP:     net.IPv4(192, 168, 1, 11),
		ConcIP:     net.IPv4(192, 168, 1, 1),
		VnicMAC:    macSeed,
		Subnet:     *net2,
		SubnetKey:  0xF,
		VnicID:     "vuuidseed",
		InstanceID: "iuuidseed",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

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

	if vnic, ssntpEvent, err := cn.CreateVnic(vnicCfgSeed); err != nil {
		b.Error("cn.CreateVnic seed failed", err, vnic, ssntpEvent)
	}

	for i := 0; i < b.N; i++ {

		if vnic, ssntpEvent, err := cn.CreateVnic(vnicCfg); err != nil {
			b.Error("cn.CreateVnic failed", err, vnic)
		} else {
			if ssntpEvent != nil {
				b.Error("cn.CreateVnic unexpected event", vnic, ssntpEvent)
			}
		}

		if ssntpEvent, err := cn.DestroyVnic(vnicCfg); err != nil {
			b.Error("cn.DestroyVnic failed", err)
		} else {
			if ssntpEvent != nil {
				b.Error("cn.DestroyVnic unexpected event", ssntpEvent)
			}
		}
	}

	if ssntpEvent, err := cn.DestroyVnic(vnicCfgSeed); err != nil {
		b.Error("cn.DestroyVnic seed failed", err, ssntpEvent)
	}
}
