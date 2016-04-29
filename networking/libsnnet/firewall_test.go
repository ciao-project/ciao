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
	"os"
	"os/exec"
	"testing"

	"github.com/01org/ciao/networking/libsnnet"
)

var fwIf, fwIfInt string

func fwinit() {
	fwIf = os.Getenv("FWIF_ENV")

	if fwIf == "" {
		fwIf = "eth0"
	}

	fwIfInt = os.Getenv("FWIFINT_ENV")

	if fwIfInt == "" {
		fwIfInt = "eth1"
	}
}

//Test firewall init for CNCI
//
//Performs basic checks of firewall primitives
//Failure indicates problem with underlying dependencies
//which could be iptables or nftables
//
//Test should pass
func TestFw_Init(t *testing.T) {
	fwinit()
	fw, err := libsnnet.InitFirewall(fwIf)

	if err != nil {
		t.Fatalf("Error: InitFirewall %v %v %v", fwIf, err, fw)
	}

	err = fw.ShutdownFirewall()
	if err != nil {
		t.Errorf("Error: Unable to shutdown firewall %v", err)
	}
}

//Tests SSH port forwarding primitives
//
//Tests the primitives used by CNCI to setup/teardown port forwarding
//
//Test should pass
func TestFw_Ssh(t *testing.T) {
	fwinit()
	fw, err := libsnnet.InitFirewall(fwIf)
	if err != nil {
		t.Fatalf("Error: InitFirewall %v %v %v", fwIf, err, fw)
	}

	err = fw.ExtPortAccess(libsnnet.FwEnable, "tcp", fwIf, 12345,
		net.ParseIP("192.168.0.101"), 22)

	if err != nil {
		t.Errorf("Error: ssh fwd failed %v", err)
	}

	err = fw.ExtPortAccess(libsnnet.FwDisable, "tcp", fwIf, 12345,
		net.ParseIP("192.168.0.101"), 22)

	if err != nil {
		t.Errorf("Error: ssh fwd disable failed %v", err)
	}

	err = fw.ShutdownFirewall()
	if err != nil {
		t.Errorf("Error: Unable to shutdown firewall %v", err)
	}
}

//Tests setting up NAT
//
//Test check if a NAT rule can be setup to perform outbound
//NAT from a given internal interface to a specified
//external interface (which has a dynamic IP, i.e DHCP)
//
//Test is expected to pass
func TestFw_Nat(t *testing.T) {
	fwinit()
	fw, err := libsnnet.InitFirewall(fwIf)
	if err != nil {
		t.Fatalf("Error: InitFirewall %v %v %v", fwIf, err, fw)
	}

	err = fw.ExtFwding(libsnnet.FwEnable, fwIf, fwIfInt)
	if err != nil {
		t.Errorf("Error: NAT failed %v", err)
	}

	err = fw.ExtFwding(libsnnet.FwDisable, fwIf, fwIfInt)
	if err != nil {
		t.Errorf("Error: NAT disable failed %v", err)
	}

	err = fw.ShutdownFirewall()
	if err != nil {
		t.Errorf("Error: Unable to shutdown firewall %v", err)
	}
}

/*
//Not fully implemented
//
//Not fully implemented
//
//Expected to pass
func TestFw_PublicIP(t *testing.T) {
	fwinit()
	fw, err := libsnnet.InitFirewall(fwIf)
	if err != nil {
		t.Fatalf("Error: InitFirewall %v %v %v", fwIf, err, fw)
	}

	intIP := net.ParseIP("192.168.0.101")
	pubIP := net.ParseIP("192.168.0.131")

	err = fw.PublicIPAccess(libsnnet.FwEnable, intIP, pubIP, fwIfInt)
	if err != nil {
		t.Errorf("%v", err)
	}

	//t.Logf("%s", libsnnet.DumpIPTables())

	err = fw.PublicIPAccess(libsnnet.FwDisable, intIP, pubIP, fwIfInt)
	if err != nil {
		t.Errorf("%v", err)
	}

	err = fw.ShutdownFirewall()
	if err != nil {
		t.Errorf("Error: Unable to shutdown firewall %v", err)
	}
}
*/

//Exercises all valid CNCI Firewall APIs
//
//This tests performs the sequence of operations typically
//performed by a CNCI Agent.
//
//Test is expected to pass
func TestFw_All(t *testing.T) {
	fwinit()
	fw, err := libsnnet.InitFirewall(fwIf)
	if err != nil {
		t.Fatalf("Error: InitFirewall %v %v %v", fwIf, err, fw)
	}

	err = fw.ExtFwding(libsnnet.FwEnable, fwIf, fwIfInt)
	if err != nil {
		t.Errorf("Error: NAT failed %v", err)
	}

	err = fw.ExtPortAccess(libsnnet.FwEnable, "tcp", fwIf, 12345,
		net.ParseIP("192.168.0.101"), 22)

	if err != nil {
		t.Errorf("Error: ssh fwd failed %v", err)
	}

	//t.Logf("%s", libsnnet.DumpIPTables())

	procIPFwd := "/proc/sys/net/ipv4/ip_forward"
	out, err := exec.Command("cat", procIPFwd).CombinedOutput()

	if err != nil {
		t.Errorf("unable to dump ip_forward %v", err)
	} else {
		t.Logf("ip_forward =[%s]", string(out))
	}

	err = fw.ExtPortAccess(libsnnet.FwDisable, "tcp", fwIf, 12345,
		net.ParseIP("192.168.0.101"), 22)

	if err != nil {
		t.Errorf("Error: ssh fwd disable failed %v", err)
	}

	err = fw.ExtFwding(libsnnet.FwDisable, fwIf, fwIfInt)
	if err != nil {
		t.Errorf("Error: NAT disable failed %v", err)
	}

	err = fw.ShutdownFirewall()
	if err != nil {
		t.Errorf("Error: Unable to shutdown firewall %v", err)
	}
}
