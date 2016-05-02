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
	"os"
	"os/exec"
	"strings"
	"testing"
)

var cnConNetEnv string
var cnDebug = false

func cnConInit() {
	cnConNetEnv = os.Getenv("SNNET_ENV")

	if cnConNetEnv == "" {
		cnConNetEnv = "192.168.0.0/24"
	}

	debug := os.Getenv("SNNET_DEBUG")
	if debug != "" && debug != "false" {
		cnDebug = true
	}
}

func debugPrint(t *testing.T, args ...interface{}) {
	if cnDebug {
		t.Log(args)
	}
}

func linkDump(t *testing.T) error {
	out, err := exec.Command("ip", "-d", "link").CombinedOutput()

	if err != nil {
		t.Errorf("unable to dump link %v", err)
	} else {
		debugPrint(t, "dumping link info \n", string(out))
	}

	return err
}

func dockerRestart(t *testing.T) error {
	out, err := exec.Command("service", "docker", "restart").CombinedOutput()
	if err != nil {
		out, err = exec.Command("systemctl", "restart", "docker").CombinedOutput()
		if err != nil {
			t.Error("docker restart", err)
		}
	}
	debugPrint(t, "docker restart\n", string(out))
	return err
}

//Will be replaced by Docker API's in launcher
//docker run -it --net=<subnet.Name> --ip=<instance.IP> --mac-address=<instance.MacAddresss>
//debian ip addr show eth0 scope global
func dockerRunVerify(t *testing.T, name string, ip net.IP, mac net.HardwareAddr, subnetID string) error {
	cmd := exec.Command("docker", "run", "--name", ip.String(), "--net="+subnetID,
		"--ip="+ip.String(), "--mac-address="+mac.String(),
		"debian", "ip", "addr", "show", "eth0", "scope", "global")
	out, err := cmd.CombinedOutput()

	if err != nil {
		t.Error("docker run failed", cmd, err)
	} else {
		debugPrint(t, "docker run dump \n", string(out))
	}

	if !strings.Contains(string(out), ip.String()) {
		t.Error("docker container IP not setup ", ip.String())
	}
	if !strings.Contains(string(out), mac.String()) {
		t.Error("docker container MAC not setup ", mac.String())
	}
	if !strings.Contains(string(out), "mtu 1400") {
		t.Error("docker container MTU not setup ")
	}

	if err := dockerContainerInfo(t, name); err != nil {
		t.Error("docker container inspect failed", name, err.Error())
	}
	return err
}

func dockerContainerDelete(t *testing.T, name string) error {
	out, err := exec.Command("docker", "stop", name).CombinedOutput()
	if err != nil {
		t.Error("docker container stop failed", name, err)
	} else {
		debugPrint(t, "docker container stop= \n", string(out))
	}

	out, err = exec.Command("docker", "rm", name).CombinedOutput()
	if err != nil {
		t.Error("docker container delete failed", name, err)
	} else {
		debugPrint(t, "docker container delete= \n", string(out))
	}
	return err
}

func dockerContainerInfo(t *testing.T, name string) error {
	out, err := exec.Command("docker", "ps", "-a").CombinedOutput()
	if err != nil {
		t.Error("docker ps -a", err)
	} else {
		debugPrint(t, "docker =\n", string(out))
	}

	out, err = exec.Command("docker", "inspect", name).CombinedOutput()
	if err != nil {
		t.Error("docker network inspect", name, err)
	} else {
		debugPrint(t, "docker network inspect \n", string(out))
	}
	return err
}

//Will be replaced by Docker API's in launcher
// docker network create -d=ciao [--ipam-driver=ciao]
// --subnet=<ContainerInfo.Subnet> --gateway=<ContainerInfo.Gate
// --opt "bridge"=<ContainerInfo.Bridge> ContainerInfo.SubnetID
//The IPAM driver needs top of the tree Docker (which needs special build)
//is not tested yet
func dockerNetCreate(t *testing.T, subnet net.IPNet, gw net.IP, bridge string, subnetID string) error {
	cmd := exec.Command("docker", "network", "create", "-d=ciao",
		"--subnet="+subnet.String(), "--gateway="+gw.String(),
		"--opt", "bridge="+bridge, subnetID)

	out, err := cmd.CombinedOutput()

	if err != nil {
		t.Error("docker network create failed", err)
	} else {
		debugPrint(t, "docker network create \n", string(out))
	}
	return err
}

//Will be replaced by Docker API's in launcher
// docker network rm ContainerInfo.SubnetID
func dockerNetDelete(t *testing.T, subnetID string) error {
	out, err := exec.Command("docker", "network", "rm", subnetID).CombinedOutput()
	if err != nil {
		t.Error("docker network delete failed", err)
	} else {
		debugPrint(t, "docker network delete=", string(out))
	}
	return err
}
func dockerNetList(t *testing.T) error {
	out, err := exec.Command("docker", "network", "ls").CombinedOutput()
	if err != nil {
		t.Error("docker network ls", err)
	} else {
		debugPrint(t, "docker network ls= \n", string(out))
	}
	return err
}

func dockerNetInfo(t *testing.T, subnetID string) error {
	out, err := exec.Command("docker", "network", "inspect", subnetID).CombinedOutput()
	if err != nil {
		t.Error("docker network inspect", err)
	} else {
		debugPrint(t, "docker network inspect=", string(out))
	}
	return err
}

//Tests typical sequence of CN Container APIs
//
//This tests exercises the standard set of APIs that
//the launcher invokes when setting up a CN and creating
//Container VNICs.
//
//Test is expected to pass
func TestCNContainer_Base(t *testing.T) {
	cn := &ComputeNode{}

	cn.NetworkConfig = &NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          GreTunnel,
	}

	cn.ID = "cnuuid"
	cnConInit()
	_, mnet, _ := net.ParseCIDR(cnConNetEnv)

	mgtNet := []net.IPNet{*mnet}
	cn.ManagementNet = mgtNet
	cn.ComputeNet = mgtNet

	if err := cn.Init(); err != nil {
		t.Fatal("ERROR: cn.Init failed", err)
	}

	if err := cn.DbRebuild(nil); err != nil {
		t.Fatal("ERROR: cn.dbRebuild failed")
	}

	dockerPlugin := NewDockerPlugin()
	if err := dockerPlugin.Init(); err != nil {
		t.Fatal("ERROR: Docker Init failed ", err)
	}

	if err := dockerPlugin.Start(); err != nil {
		t.Fatal("ERROR: Docker start failed ", err)
	}

	//Restarting docker here so the plugin will
	//be picked up without modifying the boot scripts
	if err := dockerRestart(t); err != nil {
		t.Fatal("ERROR: Docker restart failed ", err)
	}

	//From YAML on instance init
	//Two VNICs on the same tenant subnet
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	mac2, _ := net.ParseMAC("CA:FE:00:02:02:03")
	_, tnet, _ := net.ParseCIDR("192.168.111.0/24")
	tip := net.ParseIP("192.168.111.100")
	tip2 := net.ParseIP("192.168.111.102")
	cip := net.ParseIP("192.168.200.200")

	vnicCfg := &VnicConfig{
		VnicRole:   TenantContainer,
		VnicIP:     tip,
		ConcIP:     cip,
		VnicMAC:    mac,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	vnicCfg2 := &VnicConfig{
		VnicRole:   TenantContainer,
		VnicIP:     tip2,
		ConcIP:     cip,
		VnicMAC:    mac2,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid2",
		InstanceID: "iuuid2",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	var subnetID, iface string //Used to check that they match

	// Create a VNIC: Should create bridge and tunnels
	if vnic, ssntpEvent, cInfo, err := cn.CreateVnic(vnicCfg); err != nil {
		t.Error(err)
	} else {
		switch {
		case ssntpEvent == nil:
			t.Error("ERROR: expected SSNTP Event")
		case ssntpEvent.Event != SsntpTunAdd:
			t.Errorf("ERROR: cn.CreateVnic event population errror %v ", err)
		case cInfo == nil:
			t.Error("ERROR: expected Container Event")
		case cInfo.CNContainerEvent != ContainerNetworkAdd:
			t.Error("ERROR: Expected network add", ssntpEvent, cInfo)
		case cInfo.SubnetID == "":
			t.Error("ERROR: expected Container SubnetID")
		case cInfo.Subnet.String() == "":
			t.Error("ERROR: expected Container Subnet")
		case cInfo.Gateway.String() == "":
			t.Error("ERROR: expected Container Gateway")
		case cInfo.Bridge == "":
			t.Error("ERROR: expected Container Bridge")
		}
		if err := validSsntpEvent(ssntpEvent, vnicCfg); err != nil {
			t.Error("ERROR: cn.CreateVnic event population errror", vnic, ssntpEvent)
		}

		//Cache the first subnet ID we see. All subsequent should have the same
		subnetID = cInfo.SubnetID
		iface = vnic.interfaceName()
		if iface == "" {
			t.Error("ERROR: cn.CreateVnic interface population errror", vnic)
		}

		//Launcher will attach to this name and send out the event
		//Launcher will also create the logical docker network
		debugPrint(t, "VNIC created =", vnic.LinkName, ssntpEvent, cInfo)

		if err := linkDump(t); err != nil {
			t.Errorf("unable to dump link %v", err)
		}

		//Now kick off the docker commands
		if err := dockerNetCreate(t, cInfo.Subnet, cInfo.Gateway,
			cInfo.Bridge, cInfo.SubnetID); err != nil {
			t.Error("ERROR: docker network", cInfo, err)
		}
		if err := dockerNetInfo(t, cInfo.SubnetID); err != nil {
			t.Error("ERROR: docker network info", cInfo, err)
		}
		if err := dockerRunVerify(t, vnicCfg.VnicIP.String(), vnicCfg.VnicIP, vnicCfg.VnicMAC, cInfo.SubnetID); err != nil {
			t.Error("ERROR: docker run", cInfo, err)
		}
		if err := dockerContainerDelete(t, vnicCfg.VnicIP.String()); err != nil {
			t.Error("ERROR: docker network delete", cInfo, err)
		}
	}

	//Duplicate VNIC creation
	if vnic, ssntpEvent, cInfo, err := cn.CreateVnic(vnicCfg); err != nil {
		t.Error(err)
	} else {
		switch {
		case ssntpEvent != nil:
			t.Error("ERROR: DUP unexpected SSNTP event", vnic, ssntpEvent)
		case cInfo == nil:
			t.Error("ERROR: DUP expected Container Info", vnic)
		case cInfo.SubnetID != subnetID:
			t.Error("ERROR: DUP SubnetID mismatch", ssntpEvent, cInfo)
		case cInfo.CNContainerEvent != ContainerNetworkInfo:
			t.Error("ERROR: DUP Expected network info", ssntpEvent, cInfo)
		case iface != vnic.interfaceName():
			t.Errorf("ERROR: DUP interface mismatch [%v] [%v]", iface, vnic.interfaceName())
		}
	}

	//Second VNIC creation - Should succeed
	if vnic, ssntpEvent, cInfo, err := cn.CreateVnic(vnicCfg2); err != nil {
		t.Error(err)
	} else {
		switch {
		case ssntpEvent != nil:
			t.Error("ERROR: VNIC2 unexpected SSNTP event", vnic, ssntpEvent)
		case cInfo == nil:
			t.Error("ERROR: VNIC2 expected Container Info", vnic)
		case cInfo.SubnetID != subnetID:
			t.Error("ERROR: VNIC2 SubnetID mismatch", ssntpEvent, cInfo)
		case cInfo.CNContainerEvent != ContainerNetworkInfo:
			t.Error("ERROR: VNIC2 Expected network info", ssntpEvent, cInfo)
		}

		debugPrint(t, "VNIC2 created =", vnic.LinkName, ssntpEvent, cInfo)

		iface = vnic.interfaceName()
		if iface == "" {
			t.Error("ERROR: cn.CreateVnic interface population errror", vnic)
		}

		if err := linkDump(t); err != nil {
			t.Errorf("unable to dump link %v", err)
		}

		if err := dockerRunVerify(t, vnicCfg2.VnicIP.String(), vnicCfg2.VnicIP,
			vnicCfg2.VnicMAC, cInfo.SubnetID); err != nil {
			t.Error("ERROR: docker run", cInfo, err)
		}
		if err := dockerContainerDelete(t, vnicCfg2.VnicIP.String()); err != nil {
			t.Error("ERROR: docker network delete", cInfo, err)
		}
	}

	//Duplicate VNIC creation
	if vnic, ssntpEvent, cInfo, err := cn.CreateVnic(vnicCfg2); err != nil {
		t.Error(err)
	} else {
		switch {
		case ssntpEvent != nil:
			t.Error("ERROR: DUP unexpected SSNTP event", vnic, ssntpEvent)
		case cInfo == nil:
			t.Error("ERROR: DUP expected Container Info", vnic)
		case cInfo.SubnetID != subnetID:
			t.Error("ERROR: DUP SubnetID mismatch", ssntpEvent, cInfo)
		case cInfo.CNContainerEvent != ContainerNetworkInfo:
			t.Error("ERROR: DUP Expected network info", ssntpEvent, cInfo)
		case iface != vnic.interfaceName():
			t.Errorf("ERROR: DUP interface mismatch [%v] [%v]", iface, vnic.interfaceName())
		}
	}

	//Destroy the first one
	if ssntpEvent, cInfo, err := cn.DestroyVnic(vnicCfg); err != nil {
		t.Error(err)
	} else {
		switch {
		case ssntpEvent != nil:
			t.Error("ERROR: DELETE unexpected SSNTP Event", ssntpEvent)
		case cInfo != nil:
			t.Error("ERROR: DELETE unexpected Container Event")
		}
		debugPrint(t, "VNIC deleted event", ssntpEvent, cInfo)
	}

	//Destroy it again
	if ssntpEvent, cInfo, err := cn.DestroyVnic(vnicCfg); err != nil {
		t.Error(err)
	} else {
		switch {
		case ssntpEvent != nil:
			t.Error("ERROR: DELETE unexpected SSNTP Event", ssntpEvent)
		case cInfo != nil:
			t.Error("ERROR: DELETE unexpected Container event", cInfo)
		}
		debugPrint(t, "VNIC deleted event", ssntpEvent, cInfo)
	}

	// Try and destroy - should work - cInfo should be reported
	if ssntpEvent, cInfo, err := cn.DestroyVnic(vnicCfg2); err != nil {
		t.Error(err)
	} else {
		switch {
		case ssntpEvent == nil:
			t.Error("ERROR: DELETE expected SSNTP Event")
		case cInfo == nil:
			t.Error("ERROR: DELETE expected Container Event")
		case cInfo.SubnetID != subnetID:
			t.Error("ERROR: DELETE SubnetID mismatch", ssntpEvent, cInfo)
		case cInfo.CNContainerEvent != ContainerNetworkDel:
			t.Error("ERROR: DELETE Expected network delete", ssntpEvent, cInfo)
		}
		debugPrint(t, "VNIC deleted event", ssntpEvent, cInfo)

		if err := linkDump(t); err != nil {
			t.Errorf("unable to dump link %v", err)
		}
	}

	//Has to be called after the VNIC has been deleted
	if err := dockerNetDelete(t, subnetID); err != nil {
		t.Error("ERROR:", subnetID, err)
	}
	if err := dockerNetList(t); err != nil {
		t.Error("ERROR:", err)
	}

	//Destroy it again
	if ssntpEvent, cInfo, err := cn.DestroyVnic(vnicCfg2); err != nil {
		t.Error(err)
	} else {
		switch {
		case ssntpEvent != nil:
			t.Error("ERROR: unexpected SSNTP Event", ssntpEvent)
		case cInfo != nil:
			t.Error("ERROR: unexpected Container event", cInfo)
		}
		debugPrint(t, "VNIC deleted event", ssntpEvent, cInfo)
	}

	if err := dockerPlugin.Stop(); err != nil {
		t.Fatal("ERROR: Docker stop failed ", err)
	}

	if err := dockerPlugin.Close(); err != nil {
		t.Fatal("ERROR: Docker close failed ", err)
	}

}

func dockerRunTop(t *testing.T, name string, ip net.IP, mac net.HardwareAddr, subnetID string) error {
	cmd := exec.Command("docker", "run", "--name", ip.String(), "--net="+subnetID,
		"--ip="+ip.String(), "--mac-address="+mac.String(),
		"debian", "top", "-b", "-d1")
	go cmd.Run() // Ensures that the containers stays alive. Kludgy
	return nil
}

func dockerRunPingVerify(t *testing.T, name string, ip net.IP, mac net.HardwareAddr, subnetID string, addr string) error {
	cmd := exec.Command("docker", "run", "--name", ip.String(), "--net="+subnetID,
		"--ip="+ip.String(), "--mac-address="+mac.String(),
		"debian", "ping", "-c", "1", "192.168.111.100")
	out, err := cmd.CombinedOutput()

	if err != nil {
		t.Error("docker run failed", cmd, err)
	} else {
		debugPrint(t, "docker run dump \n", string(out))
	}

	if !strings.Contains(string(out), "1 packets received") {
		t.Error("docker connectivity test failed", ip.String())
	}
	return nil
}

//Tests connectivity between two node local Containers
//
//Tests connectivity between two node local Containers
//using ping between a long running container and
//container that does ping
//
//Test is expected to pass
func TestCNContainer_Connectivity(t *testing.T) {
	cn := &ComputeNode{}

	cn.NetworkConfig = &NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          GreTunnel,
	}

	cn.ID = "cnuuid"
	cnConInit()
	_, mnet, _ := net.ParseCIDR(cnConNetEnv)

	mgtNet := []net.IPNet{*mnet}
	cn.ManagementNet = mgtNet
	cn.ComputeNet = mgtNet

	if err := cn.Init(); err != nil {
		t.Fatal("ERROR: cn.Init failed", err)
	}

	if err := cn.DbRebuild(nil); err != nil {
		t.Fatal("ERROR: cn.dbRebuild failed")
	}

	dockerPlugin := NewDockerPlugin()
	if err := dockerPlugin.Init(); err != nil {
		t.Fatal("ERROR: Docker Init failed ", err)
	}

	if err := dockerPlugin.Start(); err != nil {
		t.Fatal("ERROR: Docker start failed ", err)
	}

	//Restarting docker here so the plugin will
	//be picked up without modifying the boot scripts
	if err := dockerRestart(t); err != nil {
		t.Fatal("ERROR: Docker restart failed ", err)
	}

	//From YAML on instance init
	//Two VNICs on the same tenant subnet
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	mac2, _ := net.ParseMAC("CA:FE:00:02:02:03")
	_, tnet, _ := net.ParseCIDR("192.168.111.0/24")
	tip := net.ParseIP("192.168.111.100")
	tip2 := net.ParseIP("192.168.111.102")
	cip := net.ParseIP("192.168.200.200")

	vnicCfg := &VnicConfig{
		VnicRole:   TenantContainer,
		VnicIP:     tip,
		ConcIP:     cip,
		VnicMAC:    mac,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	vnicCfg2 := &VnicConfig{
		VnicRole:   TenantContainer,
		VnicIP:     tip2,
		ConcIP:     cip,
		VnicMAC:    mac2,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid2",
		InstanceID: "iuuid2",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	_, _, cInfo, err := cn.CreateVnic(vnicCfg)
	if err != nil {
		t.Error(err)
	}

	err = dockerNetCreate(t, cInfo.Subnet, cInfo.Gateway, cInfo.Bridge, cInfo.SubnetID)
	if err != nil {
		t.Error(err)
	}

	//Kick off a long running container
	dockerRunTop(t, vnicCfg.VnicIP.String(), vnicCfg.VnicIP, vnicCfg.VnicMAC, cInfo.SubnetID)

	_, _, cInfo2, err := cn.CreateVnic(vnicCfg2)
	if err != nil {
		t.Error(err)
	}

	if err := dockerRunPingVerify(t, vnicCfg2.VnicIP.String(), vnicCfg2.VnicIP,
		vnicCfg2.VnicMAC, cInfo2.SubnetID, vnicCfg.VnicIP.String()); err != nil {
		t.Error(err)
	}

	//Destroy the containers
	if err := dockerContainerDelete(t, vnicCfg.VnicIP.String()); err != nil {
		t.Error(err)
	}
	if err := dockerContainerDelete(t, vnicCfg2.VnicIP.String()); err != nil {
		t.Error(err)
	}

	//Destroy the VNICs
	if _, _, err := cn.DestroyVnic(vnicCfg); err != nil {
		t.Error(err)
	}
	if _, _, err := cn.DestroyVnic(vnicCfg2); err != nil {
		t.Error(err)
	}

	//Destroy the network, has to be called after the VNIC has been deleted
	if err := dockerNetDelete(t, cInfo.SubnetID); err != nil {
		t.Error(err)
	}
	if err := dockerPlugin.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := dockerPlugin.Close(); err != nil {
		t.Fatal(err)
	}
}

//Tests VM and Container VNIC Interop
//
//Tests that VM and Container VNICs can co-exist
//by created VM and Container VNICs in different orders and in each case
//tests that the Network Connectivity is functional
//
//Test is expected to pass
func TestCNContainer_Interop1(t *testing.T) {
	cn := &ComputeNode{}

	cn.NetworkConfig = &NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          GreTunnel,
	}

	cn.ID = "cnuuid"
	cnConInit()
	_, mnet, _ := net.ParseCIDR(cnConNetEnv)

	mgtNet := []net.IPNet{*mnet}
	cn.ManagementNet = mgtNet
	cn.ComputeNet = mgtNet

	if err := cn.Init(); err != nil {
		t.Fatal("ERROR: cn.Init failed", err)
	}

	if err := cn.DbRebuild(nil); err != nil {
		t.Fatal("ERROR: cn.dbRebuild failed")
	}

	dockerPlugin := NewDockerPlugin()
	if err := dockerPlugin.Init(); err != nil {
		t.Fatal("ERROR: Docker Init failed ", err)
	}

	if err := dockerPlugin.Start(); err != nil {
		t.Fatal("ERROR: Docker start failed ", err)
	}

	//Restarting docker here so the plugin will
	//be picked up without modifying the boot scripts
	if err := dockerRestart(t); err != nil {
		t.Fatal("ERROR: Docker restart failed ", err)
	}

	//From YAML on instance init
	//Two VNICs on the same tenant subnet
	_, tnet, _ := net.ParseCIDR("192.168.111.0/24")
	tip := net.ParseIP("192.168.111.100")
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	cip := net.ParseIP("192.168.200.200")
	mac2, _ := net.ParseMAC("CA:FE:00:02:02:03")
	tip2 := net.ParseIP("192.168.111.102")
	mac3, _ := net.ParseMAC("CA:FE:00:03:02:03")
	tip3 := net.ParseIP("192.168.111.103")
	mac4, _ := net.ParseMAC("CA:FE:00:04:02:03")
	tip4 := net.ParseIP("192.168.111.104")

	vnicCfg := &VnicConfig{
		VnicRole:   TenantContainer,
		VnicIP:     tip,
		ConcIP:     cip,
		VnicMAC:    mac,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	vnicCfg2 := &VnicConfig{
		VnicRole:   TenantContainer,
		VnicIP:     tip2,
		ConcIP:     cip,
		VnicMAC:    mac2,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid2",
		InstanceID: "iuuid2",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	vnicCfg3 := &VnicConfig{
		VnicRole:   TenantVM,
		VnicIP:     tip3,
		ConcIP:     cip,
		VnicMAC:    mac3,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid3",
		InstanceID: "iuuid3",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	vnicCfg4 := &VnicConfig{
		VnicRole:   TenantVM,
		VnicIP:     tip4,
		ConcIP:     cip,
		VnicMAC:    mac4,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid4",
		InstanceID: "iuuid4",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	_, _, cInfo, err := cn.CreateVnic(vnicCfg)
	if err != nil {
		t.Error(err)
	}

	err = dockerNetCreate(t, cInfo.Subnet, cInfo.Gateway, cInfo.Bridge, cInfo.SubnetID)
	if err != nil {
		t.Error(err)
	}

	_, _, _, err = cn.CreateVnic(vnicCfg3)
	if err != nil {
		t.Error(err)
	}

	//Kick off a long running container
	dockerRunTop(t, vnicCfg.VnicIP.String(), vnicCfg.VnicIP, vnicCfg.VnicMAC, cInfo.SubnetID)

	_, _, cInfo2, err := cn.CreateVnic(vnicCfg2)
	if err != nil {
		t.Error(err)
	}

	_, _, _, err = cn.CreateVnic(vnicCfg4)
	if err != nil {
		t.Error(err)
	}

	if err := dockerRunPingVerify(t, vnicCfg2.VnicIP.String(), vnicCfg2.VnicIP,
		vnicCfg2.VnicMAC, cInfo2.SubnetID, vnicCfg.VnicIP.String()); err != nil {
		t.Error(err)
	}

	//Destroy the containers
	if err := dockerContainerDelete(t, vnicCfg.VnicIP.String()); err != nil {
		t.Error(err)
	}
	if err := dockerContainerDelete(t, vnicCfg2.VnicIP.String()); err != nil {
		t.Error(err)
	}

	if _, _, err := cn.DestroyVnic(vnicCfg); err != nil {
		t.Error(err)
	}
	if _, _, err := cn.DestroyVnic(vnicCfg2); err != nil {
		t.Error(err)
	}
	if _, _, err := cn.DestroyVnic(vnicCfg3); err != nil {
		t.Error(err)
	}
	if err := dockerNetDelete(t, cInfo.SubnetID); err != nil {
		t.Error(err)
	}
	if _, _, err := cn.DestroyVnic(vnicCfg4); err != nil {
		t.Error(err)
	}
	if err := dockerPlugin.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := dockerPlugin.Close(); err != nil {
		t.Fatal(err)
	}
}

//Tests VM and Container VNIC Interop
//
//Tests that VM and Container VNICs can co-exist
//by created VM and Container VNICs in different orders and in each case
//tests that the Network Connectivity is functional
//
//Test is expected to pass
func TestCNContainer_Interop2(t *testing.T) {
	cn := &ComputeNode{}

	cn.NetworkConfig = &NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          GreTunnel,
	}

	cn.ID = "cnuuid"
	cnConInit()
	_, mnet, _ := net.ParseCIDR(cnConNetEnv)

	mgtNet := []net.IPNet{*mnet}
	cn.ManagementNet = mgtNet
	cn.ComputeNet = mgtNet

	if err := cn.Init(); err != nil {
		t.Fatal("ERROR: cn.Init failed", err)
	}

	if err := cn.DbRebuild(nil); err != nil {
		t.Fatal("ERROR: cn.dbRebuild failed")
	}

	dockerPlugin := NewDockerPlugin()
	if err := dockerPlugin.Init(); err != nil {
		t.Fatal("ERROR: Docker Init failed ", err)
	}

	if err := dockerPlugin.Start(); err != nil {
		t.Fatal("ERROR: Docker start failed ", err)
	}

	//Restarting docker here so the plugin will
	//be picked up without modifying the boot scripts
	if err := dockerRestart(t); err != nil {
		t.Fatal("ERROR: Docker restart failed ", err)
	}

	//From YAML on instance init
	//Two VNICs on the same tenant subnet
	_, tnet, _ := net.ParseCIDR("192.168.111.0/24")
	tip := net.ParseIP("192.168.111.100")
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	cip := net.ParseIP("192.168.200.200")
	mac2, _ := net.ParseMAC("CA:FE:00:02:02:03")
	tip2 := net.ParseIP("192.168.111.102")
	mac3, _ := net.ParseMAC("CA:FE:00:03:02:03")
	tip3 := net.ParseIP("192.168.111.103")
	mac4, _ := net.ParseMAC("CA:FE:00:04:02:03")
	tip4 := net.ParseIP("192.168.111.104")

	vnicCfg := &VnicConfig{
		VnicRole:   TenantContainer,
		VnicIP:     tip,
		ConcIP:     cip,
		VnicMAC:    mac,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	vnicCfg2 := &VnicConfig{
		VnicRole:   TenantContainer,
		VnicIP:     tip2,
		ConcIP:     cip,
		VnicMAC:    mac2,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid2",
		InstanceID: "iuuid2",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	vnicCfg3 := &VnicConfig{
		VnicRole:   TenantVM,
		VnicIP:     tip3,
		ConcIP:     cip,
		VnicMAC:    mac3,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid3",
		InstanceID: "iuuid3",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	vnicCfg4 := &VnicConfig{
		VnicRole:   TenantVM,
		VnicIP:     tip4,
		ConcIP:     cip,
		VnicMAC:    mac4,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid4",
		InstanceID: "iuuid4",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	_, _, _, err := cn.CreateVnic(vnicCfg3)
	if err != nil {
		t.Error(err)
	}
	_, _, cInfo, err := cn.CreateVnic(vnicCfg)
	if err != nil {
		t.Error(err)
	}

	err = dockerNetCreate(t, cInfo.Subnet, cInfo.Gateway, cInfo.Bridge, cInfo.SubnetID)
	if err != nil {
		t.Error(err)
	}

	//Kick off a long running container
	dockerRunTop(t, vnicCfg.VnicIP.String(), vnicCfg.VnicIP, vnicCfg.VnicMAC, cInfo.SubnetID)

	_, _, cInfo2, err := cn.CreateVnic(vnicCfg2)
	if err != nil {
		t.Error(err)
	}
	if err := dockerRunPingVerify(t, vnicCfg2.VnicIP.String(), vnicCfg2.VnicIP,
		vnicCfg2.VnicMAC, cInfo2.SubnetID, vnicCfg.VnicIP.String()); err != nil {
		t.Error(err)
	}

	//Destroy the containers
	if err := dockerContainerDelete(t, vnicCfg.VnicIP.String()); err != nil {
		t.Error(err)
	}
	if err := dockerContainerDelete(t, vnicCfg2.VnicIP.String()); err != nil {
		t.Error(err)
	}

	_, _, _, err = cn.CreateVnic(vnicCfg4)
	if err != nil {
		t.Error(err)
	}

	//Destroy the VNICs
	if _, _, err := cn.DestroyVnic(vnicCfg); err != nil {
		t.Error(err)
	}
	if _, _, err := cn.DestroyVnic(vnicCfg2); err != nil {
		t.Error(err)
	}

	//Destroy the network, has to be called after the VNIC has been deleted
	if err := dockerNetDelete(t, cInfo.SubnetID); err != nil {
		t.Error(err)
	}
	if _, _, err := cn.DestroyVnic(vnicCfg4); err != nil {
		t.Error(err)
	}
	if _, _, err := cn.DestroyVnic(vnicCfg3); err != nil {
		t.Error(err)
	}
	if err := dockerPlugin.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := dockerPlugin.Close(); err != nil {
		t.Fatal(err)
	}
}
