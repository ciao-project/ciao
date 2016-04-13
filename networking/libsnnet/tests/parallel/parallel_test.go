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
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/01org/ciao/networking/libsnnet"
)

var cnNetEnv string
var cnParallel bool = true

//Controls the number of go routines that concurrently invoke Network APIs
//This checks that the internal throttling is working
var cnMaxOutstanding = 128

var scaleCfg = struct {
	maxBridgesShort int
	maxVnicsShort   int
	maxBridgesLong  int
	maxVnicsLong    int
}{2, 64, 200, 32}

const (
	allRoles = libsnnet.TenantContainer + libsnnet.TenantVM
)

func cninit() {
	cnNetEnv = os.Getenv("SNNET_ENV")

	if cnNetEnv == "" {
		cnNetEnv = "10.3.66.0/24"
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

func CNAPI_Parallel(t *testing.T, role libsnnet.VnicRole, modelCancel bool) {

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

			if _, _, _, err := cn.CreateVnicV2(vnicCfg); err != nil {
				if !modelCancel {
					//We expect failures only when we have cancellations
					t.Error("ERROR: cn.CreateVnicV2  failed", vnicCfg, err)
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
			if _, _, err := cn.DestroyVnicV2(vnicCfg); err != nil {
				if !modelCancel {
					//We expect failures only when we have cancellations
					t.Error("ERROR: cn.DestroyVnicV2 failed event", vnicCfg, err)
				}
			}
		}(vnicCfg)
	}

	wg.Wait()
}

func TestCNContainer_Parallel(t *testing.T) {
	CNAPI_Parallel(t, libsnnet.TenantContainer, false)
}

func TestCNVM_Parallel(t *testing.T) {
	CNAPI_Parallel(t, libsnnet.TenantVM, false)
}

func TestCNVMContainer_Parallel(t *testing.T) {
	CNAPI_Parallel(t, libsnnet.TenantContainer+libsnnet.TenantVM, false)
}

func TestCNVMContainer_Cancel(t *testing.T) {
	CNAPI_Parallel(t, libsnnet.TenantContainer+libsnnet.TenantVM, true)
}
