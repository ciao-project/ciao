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

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/01org/ciao/networking/libsnnet"
)

func main() {
	operationIn := flag.String("operation", "create", "operation <create|delete>")
	nwNodeIn := flag.Bool("nwNode", false, "true if Network Node")
	nwIn := flag.String("subnet", "", "subnet of the compute network")
	macIn := flag.String("mac", "DE:AD:BE:EF:02:03", "VNIC MAC Address")
	vnicIDIn := flag.String("vuuid", "vuuid", "VNIC UUID")
	instanceIDIn := flag.String("iuuid", "iuuid", "instance UUID")

	vnicNwIn := flag.String("vnicsubnet", "127.0.0.1/24", "subnet of vnic network")
	vnicIPIn := flag.String("vnicIP", "127.0.0.1", "VNIC IP")
	concIPIn := flag.String("cnci", "127.0.0.1", "CNCI IP")

	tenantIDIn := flag.String("tuuid", "tuuid", "tunnel UUID")
	subnetIDIn := flag.String("suuid", "suuid", "subnet UUID")
	concIDIn := flag.String("cnciuuid", "cnciuuid", "CNCI UUID")
	cnIDIn := flag.String("cnuuid", "cnuuid", "CN UUID")

	flag.Parse()

	_, vnet, err := net.ParseCIDR(*vnicNwIn)
	if err != nil {
		fmt.Println("Invalid vnic subnet ", err)
		os.Exit(-1)
	}
	subnetKey := binary.LittleEndian.Uint32(vnet.IP)

	cn := &libsnnet.ComputeNode{
		NetworkConfig: &libsnnet.NetworkConfig{
			Mode: libsnnet.GreTunnel,
		},
	}

	if *nwIn != "" {
		_, snet, err := net.ParseCIDR(*nwIn)
		if err != nil {
			fmt.Println("Invalid subnet ", err)
			os.Exit(-1)
		}
		cn.ManagementNet = []net.IPNet{*snet}
		cn.ComputeNet = []net.IPNet{*snet}
	}
	cn.ID = *cnIDIn

	if err := cn.Init(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	if err := cn.DbRebuild(nil); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	vnicIP := net.ParseIP(*vnicIPIn)
	if vnicIP == nil {
		fmt.Println("Invalid vnic IP")
		os.Exit(-1)
	}

	//Create a compute VNIC
	if !*nwNodeIn {

		concIP := net.ParseIP(*concIPIn)
		if concIP == nil {
			fmt.Println("Invalid Conc IP")
			os.Exit(-1)
		}

		//From YAML on instance init
		mac, _ := net.ParseMAC(*macIn)
		vnicCfg := &libsnnet.VnicConfig{
			VnicRole:   libsnnet.TenantVM,
			VnicIP:     vnicIP,
			ConcIP:     concIP,
			VnicMAC:    mac,
			Subnet:     *vnet,
			SubnetKey:  int(subnetKey),
			VnicID:     *vnicIDIn,
			InstanceID: *instanceIDIn,
			TenantID:   *tenantIDIn,
			SubnetID:   *subnetIDIn,
			ConcID:     *concIDIn,
		}

		switch *operationIn {
		case "create":
			fmt.Println("Creating VNIC for Workload")
			if vnic, ssntpEvent, err := cn.CreateVnic(vnicCfg); err != nil {
				fmt.Println(err)
				os.Exit(-1)
			} else {
				if ssntpEvent != nil {
					fmt.Println("SSNTP Event :=", ssntpEvent)
					fmt.Println("tap interface :=", vnic.LinkName)
				} else {
					fmt.Println("tap interface :=", vnic.LinkName)
				}
			}
		case "delete":
			fmt.Println("Deleting VNIC for Workload")
			if ssntpEvent, err := cn.DestroyVnic(vnicCfg); err != nil {
				fmt.Println(err)
				os.Exit(-1)
			} else {
				if ssntpEvent != nil {
					fmt.Println("SSNTP Event:=", ssntpEvent)
				}
			}
		}
		os.Exit(0)
	}

	//Network Node
	if *nwNodeIn {
		mac, _ := net.ParseMAC(*macIn)
		vnicCfg := &libsnnet.VnicConfig{
			VnicRole:   libsnnet.DataCenter,
			VnicMAC:    mac,
			VnicID:     *vnicIDIn,
			InstanceID: *instanceIDIn,
			TenantID:   *tenantIDIn,
		}

		switch *operationIn {
		case "create":
			if cvnic, err := cn.CreateCnciVnic(vnicCfg); err != nil {
				fmt.Println(err)
				os.Exit(-1)
			} else {
				fmt.Println("CNCI VNIC :=", cvnic)
				fmt.Println("macvtap interface :=", cvnic.LinkName)
			}
		case "delete":
			if err := cn.DestroyCnciVnic(vnicCfg); err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}
		}

	}
}
