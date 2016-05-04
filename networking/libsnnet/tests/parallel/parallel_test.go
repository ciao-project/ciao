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

package parallel

import (
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/01org/ciao/networking/libsnnet"
)

var cnNetEnv string
var cnParallel = true

//Controls the number of go routines that concurrently invoke Network APIs
//This checks that the internal throttling is working
var cnMaxOutstanding = 128

var scaleCfg = struct {
	maxBridgesShort int
	maxVnicsShort   int
	maxBridgesLong  int
	maxVnicsLong    int
}{2, 8, 200, 32}

const (
	allRoles = libsnnet.TenantContainer + libsnnet.TenantVM
)

func cninit() {
	cnNetEnv = os.Getenv("SNNET_ENV")

	if cnNetEnv == "" {
		cnNetEnv = "127.0.0.1/24"
	}

	if cnParallel {
		runtime.GOMAXPROCS(runtime.NumCPU())
	} else {
		runtime.GOMAXPROCS(1)
	}
}

func logTime(t *testing.T, start time.Time, fn string) {
	elapsedTime := time.Since(start)
	t.Logf("function %s took %s", fn, elapsedTime)
}

func CNAPIParallel(t *testing.T, role libsnnet.VnicRole, modelCancel bool) {

	var sem = make(chan int, cnMaxOutstanding)

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
	concIP := net.IPv4(192, 168, 254, 1)

	var maxBridges, maxVnics int
	if testing.Short() {
		maxBridges = scaleCfg.maxBridgesShort
		maxVnics = scaleCfg.maxVnicsShort
	} else {
		maxBridges = scaleCfg.maxBridgesLong
		maxVnics = scaleCfg.maxVnicsLong
	}

	channelSize := maxBridges*maxVnics + 1
	createCh := make(chan *libsnnet.VnicConfig, channelSize)
	destroyCh := make(chan *libsnnet.VnicConfig, channelSize)
	cancelCh := make(chan chan interface{}, channelSize)

	t.Log("Priming interfaces")

	for s3 := 1; s3 <= maxBridges; s3++ {
		s4 := 0
		_, tenantNet, _ := net.ParseCIDR("192.168." + strconv.Itoa(s3) + "." + strconv.Itoa(s4) + "/24")
		subnetID := "suuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)

		for s4 := 2; s4 <= maxVnics; s4++ {

			vnicIP := net.IPv4(192, 168, byte(s3), byte(s4))
			mac, _ := net.ParseMAC("CA:FE:00:01:02:03")

			vnicID := "vuuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)
			instanceID := "iuuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)

			role := role
			if role == allRoles {
				if s4%2 == 1 {
					role = libsnnet.TenantContainer
				} else {
					role = libsnnet.TenantVM
				}
			}

			vnicCfg := &libsnnet.VnicConfig{
				VnicRole:   role,
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

			if modelCancel {
				vnicCfg.CancelChan = make(chan interface{})
			}

			createCh <- vnicCfg
			destroyCh <- vnicCfg
			cancelCh <- vnicCfg.CancelChan
		}
	}

	close(createCh)
	close(destroyCh)
	close(cancelCh)

	var wg sync.WaitGroup
	wg.Add(len(createCh))

	if modelCancel {
		for c := range cancelCh {
			go func(c chan interface{}) {
				time.Sleep(100 * time.Millisecond)
				close(c)
			}(c)
		}
	}

	for vnicCfg := range createCh {

		sem <- 1
		go func(vnicCfg *libsnnet.VnicConfig) {
			defer wg.Done()
			defer func() {
				<-sem
			}()

			if vnicCfg == nil {
				t.Errorf("WARNING: VNIC nil")
				return
			}

			defer logTime(t, time.Now(), "Create VNIC")

			if _, _, _, err := cn.CreateVnic(vnicCfg); err != nil {
				if !modelCancel {
					//We expect failures only when we have cancellations
					t.Error("ERROR: cn.CreateVnic  failed", vnicCfg, err)
				}
			}

		}(vnicCfg)
	}

	wg.Wait()

	wg.Add(len(destroyCh))
	for vnicCfg := range destroyCh {
		sem <- 1
		go func(vnicCfg *libsnnet.VnicConfig) {
			defer wg.Done()
			defer func() {
				<-sem
			}()

			if vnicCfg == nil {
				t.Errorf("WARNING: VNIC nil")
				return
			}
			defer logTime(t, time.Now(), "Destroy VNIC")
			if _, _, err := cn.DestroyVnic(vnicCfg); err != nil {
				if !modelCancel {
					//We expect failures only when we have cancellations
					t.Error("ERROR: cn.DestroyVnic failed event", vnicCfg, err)
				}
			}
		}(vnicCfg)
	}

	wg.Wait()
}

func TestCNContainer_Parallel(t *testing.T) {
	CNAPIParallel(t, libsnnet.TenantContainer, false)
}

func TestCNVM_Parallel(t *testing.T) {
	CNAPIParallel(t, libsnnet.TenantVM, false)
}

func TestCNVMContainer_Parallel(t *testing.T) {
	CNAPIParallel(t, libsnnet.TenantContainer+libsnnet.TenantVM, false)
}

func TestCNVMContainer_Cancel(t *testing.T) {
	CNAPIParallel(t, libsnnet.TenantContainer+libsnnet.TenantVM, true)
}

//Docker Testing
//TODO: Place all docker utility functions in a single file
func dockerRestart(t *testing.T) error {
	out, err := exec.Command("service", "docker", "restart").CombinedOutput()
	if err != nil {
		out, err = exec.Command("systemctl", "restart", "docker").CombinedOutput()
		if err != nil {
			t.Error("docker restart", string(out), err)
		}
	}
	return err
}

//Will be replaced by Docker API's in launcher
//docker run -it --net=none debian ip addr show lo
func dockerRunNetNone(t *testing.T, name string, ip net.IP, mac net.HardwareAddr, subnetID string) error {
	defer logTime(t, time.Now(), "dockerRunNetNone")

	cmd := exec.Command("docker", "run", "--name", ip.String(), "--net=none",
		"debian", "ip", "addr", "show", "lo")
	out, err := cmd.CombinedOutput()

	if err != nil {
		t.Error("docker run failed", cmd, string(out), err)
	}

	if err := dockerContainerInfo(t, name); err != nil {
		t.Error("docker container inspect failed", name, err.Error())
	}

	return err
}

//Will be replaced by Docker API's in launcher
//docker run -it debian ip addr show lo
func dockerRunNetDocker(t *testing.T, name string, ip net.IP, mac net.HardwareAddr, subnetID string) error {
	defer logTime(t, time.Now(), "dockerRunNetDocker")

	cmd := exec.Command("docker", "run", "--name", ip.String(),
		"debian", "ip", "addr", "show", "lo")
	out, err := cmd.CombinedOutput()

	if err != nil {
		t.Error("docker run failed", cmd, string(out), err)
	}

	if err := dockerContainerInfo(t, name); err != nil {
		t.Error("docker container inspect failed", name, err.Error())
	}

	return err
}

//Will be replaced by Docker API's in launcher
//docker run -it --net=<subnet.Name> --ip=<instance.IP> --mac-address=<instance.MacAddresss>
//debian ip addr show eth0 scope global
func dockerRunVerify(t *testing.T, name string, ip net.IP, mac net.HardwareAddr, subnetID string) error {
	defer logTime(t, time.Now(), "dockerRunVerify")

	cmd := exec.Command("docker", "run", "--name", ip.String(), "--net="+subnetID,
		"--ip="+ip.String(), "--mac-address="+mac.String(),
		"debian", "ip", "addr", "show", "eth0", "scope", "global")
	out, err := cmd.CombinedOutput()

	if err != nil {
		t.Error("docker run failed", cmd, string(out), err)
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
	defer logTime(t, time.Now(), "dockerContainerDelete")
	out, err := exec.Command("docker", "stop", name).CombinedOutput()
	if err != nil {
		t.Error("docker container stop failed", name, string(out), err)
	}

	out, err = exec.Command("docker", "rm", name).CombinedOutput()
	if err != nil {
		t.Error("docker container delete failed", name, string(out), err)
	}
	return err
}

func dockerContainerInfo(t *testing.T, name string) error {
	defer logTime(t, time.Now(), "dockerContainerInfo")
	out, err := exec.Command("docker", "ps", "-a").CombinedOutput()
	if err != nil {
		t.Error("docker ps -a", string(out), err)
	}

	out, err = exec.Command("docker", "inspect", name).CombinedOutput()
	if err != nil {
		t.Error("docker network inspect", name, string(out), err)
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
	defer logTime(t, time.Now(), "dockerNetCreate")
	cmd := exec.Command("docker", "network", "create", "-d=ciao",
		"--subnet="+subnet.String(), "--gateway="+gw.String(),
		"--opt", "bridge="+bridge, subnetID)

	out, err := cmd.CombinedOutput()

	if err != nil {
		t.Error("docker network create failed", string(out), err)
	}
	return err
}

//Will be replaced by Docker API's in launcher
// docker network rm ContainerInfo.SubnetID
func dockerNetDelete(t *testing.T, subnetID string) error {
	defer logTime(t, time.Now(), "dockerNetDelete")
	out, err := exec.Command("docker", "network", "rm", subnetID).CombinedOutput()
	if err != nil {
		t.Error("docker network delete failed", string(out), err)
	}
	return err
}

func dockerNetList(t *testing.T) error {
	defer logTime(t, time.Now(), "dockerNetList")
	out, err := exec.Command("docker", "network", "ls").CombinedOutput()
	if err != nil {
		t.Error("docker network ls", string(out), err)
	}
	return err
}

func dockerNetInfo(t *testing.T, subnetID string) error {
	defer logTime(t, time.Now(), "dockerNetInfo")
	out, err := exec.Command("docker", "network", "inspect", subnetID).CombinedOutput()
	if err != nil {
		t.Error("docker network inspect", string(out), err)
	}
	return err
}

type dockerNetType int

const (
	netCiao dockerNetType = iota
	netDockerNone
	netDockerDefault
)

//Tests launch of Docker containers at scale (serially)
//
//This tests exercises attempts to launch large
//numbers of docker containers at scale to isolate
//any issues with plugin responsiveness
//
//Test is expected to pass
func DockerSerial(netType dockerNetType, t *testing.T) {
	defer logTime(t, time.Now(), "TestDockerSerial")
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

	dockerPlugin := libsnnet.NewDockerPlugin()
	if err := dockerPlugin.Init(); err != nil {
		t.Fatal("ERROR: Docker Init failed ", err)
	}
	defer func() { _ = dockerPlugin.Close() }()

	if err := dockerPlugin.Start(); err != nil {
		t.Fatal("ERROR: Docker start failed ", err)
	}
	defer func() { _ = dockerPlugin.Stop() }()

	//Restarting docker here so the the plugin will
	//be picked up without modifing the boot scripts
	if err := dockerRestart(t); err != nil {
		t.Fatal("ERROR: Docker restart failed ", err)
	}

	//From YAML on instance init
	tenantID := "tenantuuid"
	concIP := net.IPv4(192, 168, 254, 1)

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
		_, tenantNet, _ := net.ParseCIDR("192.168." + strconv.Itoa(s3) + "." + strconv.Itoa(s4) + "/24")
		subnetID := "suuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)

		for s4 := 2; s4 <= maxVnics; s4++ {

			vnicIP := net.IPv4(192, 168, byte(s3), byte(s4))
			mac, _ := net.ParseMAC("CA:FE:00:01:02:03")

			vnicID := "vuuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)
			instanceID := "iuuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)
			role := libsnnet.TenantContainer

			vnicCfg := &libsnnet.VnicConfig{
				VnicRole:   role,
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

			// Create a VNIC: Should create bridge and tunnels
			if _, _, cInfo, err := cn.CreateVnic(vnicCfg); err != nil {
				t.Error(err)
			} else {
				defer func() { _, _, _ = cn.DestroyVnic(vnicCfg) }()

				if cInfo.CNContainerEvent == libsnnet.ContainerNetworkAdd {
					if err := dockerNetCreate(t, cInfo.Subnet, cInfo.Gateway,
						cInfo.Bridge, cInfo.SubnetID); err != nil {
						t.Error("ERROR: docker network", cInfo, err)
					} else {
						defer func() { _ = dockerNetDelete(t, cInfo.SubnetID) }()
					}
				}

				switch netType {
				case netCiao:
					if err := dockerRunVerify(t, vnicCfg.VnicIP.String(), vnicCfg.VnicIP, vnicCfg.VnicMAC, cInfo.SubnetID); err != nil {
						t.Error("ERROR: docker run", cInfo, err)
					}
				case netDockerNone:
					if err := dockerRunNetNone(t, vnicCfg.VnicIP.String(), vnicCfg.VnicIP, vnicCfg.VnicMAC, cInfo.SubnetID); err != nil {
						t.Error("ERROR: docker run", cInfo, err)
					}
				case netDockerDefault:
					if err := dockerRunNetDocker(t, vnicCfg.VnicIP.String(), vnicCfg.VnicIP, vnicCfg.VnicMAC, cInfo.SubnetID); err != nil {
						t.Error("ERROR: docker run", cInfo, err)
					}
				}
				defer func() { _ = dockerContainerDelete(t, vnicCfg.VnicIP.String()) }()
			}
		}
	}
}

//Tests launch of Docker containers at scale (serially)
//
//This tests exercises attempts to launch large
//numbers of docker containers at scale to isolate
//any issues with plugin responsiveness
//
//Test is expected to pass
func TestDockerNetCiao_Serial(t *testing.T) {
	DockerSerial(netCiao, t)
}

//Tests launch of Docker containers at scale (serially)
//
//This tests exercises attempts to launch large
//numbers of docker containers at scale to isolate
//any issues with plugin responsiveness
//This test benchmarks docker without networking
//
//Test is expected to pass
func TestDockerNetNone_Serial(t *testing.T) {
	DockerSerial(netDockerNone, t)
}

//Tests launch of Docker containers at scale (serially)
//
//This tests exercises attempts to launch large
//numbers of docker containers at scale to isolate
//any issues with plugin responsiveness
//This test benchmarks docker with default networking
//
//Test is expected to pass
func TestDockerNetDocker_Serial(t *testing.T) {
	DockerSerial(netDockerDefault, t)
}
