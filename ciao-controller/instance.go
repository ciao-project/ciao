/*
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
*/

package main

import (
	"encoding/json"
	"fmt"
	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/payloads"
	"github.com/docker/distribution/uuid"
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
	"net"
	"time"
)

type config struct {
	sc     payloads.Start
	config string
	cnci   bool
	mac    string
	ip     string
}

type instance struct {
	types.Instance
	newConfig config
	context   *controller
	startTime time.Time
}

func isCNCIWorkload(workload *types.Workload) bool {
	for r := range workload.Defaults {
		if workload.Defaults[r].Type == payloads.NetworkNode {
			return true
		}
	}
	return false
}

func newInstance(context *controller, tenantID string, workload *types.Workload) (i *instance, err error) {
	id := uuid.Generate()

	config, err := newConfig(context, workload, id.String(), tenantID)
	if err != nil {
		return
	}

	usage := config.GetResources()

	newInstance := types.Instance{
		TenantID:   tenantID,
		WorkloadID: workload.ID,
		State:      payloads.Pending,
		ID:         id.String(),
		CNCI:       config.cnci,
		IPAddress:  config.ip,
		MACAddress: config.mac,
		Usage:      usage,
	}

	i = &instance{
		context:   context,
		newConfig: config,
		Instance:  newInstance,
	}
	return
}

func (i *instance) Add() (err error) {
	if i.CNCI == false {
		ds := i.context.ds
		go ds.AddInstance(&i.Instance)
	} else {
		i.context.ds.AddTenantCNCI(i.TenantID, i.ID, i.MACAddress)
	}
	return
}

func (i *instance) Clean() (err error) {
	if i.CNCI == false {
		i.context.ds.ReleaseTenantIP(i.TenantID, i.IPAddress)
	}
	return
}

func (i *instance) Allowed() (b bool, err error) {
	if i.CNCI == true {
		// should I bother to check the tenant id exists?
		return true, nil
	}

	ds := i.context.ds
	tenant, err := ds.GetTenant(i.TenantID)
	if err != nil {
		return false, err
	}

	for _, res := range tenant.Resources {
		// check instance count separately
		if res.Rtype == 1 {
			if res.OverLimit(1) {
				return false, nil
			}
			continue
		}
		if res.OverLimit(i.Usage[res.Rname]) {
			return false, nil
		}
	}
	return true, nil
}

func (c *config) GetResources() (resources map[string]int) {
	rr := c.sc.Start.RequestedResources

	// convert RequestedResources into a map[string]int
	resources = make(map[string]int)
	for i := range rr {
		resources[string(rr[i].Type)] = rr[i].Value
	}
	return
}

func newConfig(context *controller, wl *types.Workload, instanceID string, tenantID string) (config config, err error) {
	type UserData struct {
		UUID     string `json:"uuid"`
		Hostname string `json:"hostname"`
	}
	var userData UserData

	baseConfig := wl.Config
	defaults := wl.Defaults
	imageID := wl.ImageID
	fwType := wl.FWType

	tenant, err := context.ds.GetTenant(tenantID)
	if err != nil {
		fmt.Println("unable to get tenant")
	}

	config.cnci = isCNCIWorkload(wl)

	var networking payloads.NetworkResources

	// do we ever need to save the vnic uuid?
	networking.VnicUUID = uuid.Generate().String()

	if config.cnci == false {
		ipAddress, err := context.ds.AllocateTenantIP(tenantID)
		if err != nil {
			fmt.Println("Unable to allocate IP address: ", err)
			return config, err
		}

		networking.VnicMAC = newTenantHardwareAddr(ipAddress).String()

		// send in CIDR notation?
		networking.PrivateIP = ipAddress.String()
		config.ip = ipAddress.String()
		mask := net.IPv4Mask(255, 255, 255, 0)
		ipnet := net.IPNet{
			IP:   ipAddress.Mask(mask),
			Mask: mask,
		}
		networking.Subnet = ipnet.String()
		networking.ConcentratorUUID = tenant.CNCIID

		// in theory we should refuse to go on if ip is null
		// for now let's keep going
		networking.ConcentratorIP = tenant.CNCIIP

		// set the hostname and uuid for userdata
		userData.UUID = instanceID
		userData.Hostname = instanceID
	} else {
		networking.VnicMAC = tenant.CNCIMAC

		// set the hostname and uuid for userdata
		userData.UUID = instanceID
		userData.Hostname = "cnci-" + tenantID
	}

	// hardcode persistence until changes can be made to workload
	// template datastore.  Estimated resources can be blank
	// for now because we don't support it yet.
	startCmd := payloads.StartCmd{
		TenantUUID:          tenantID,
		InstanceUUID:        instanceID,
		ImageUUID:           imageID,
		FWType:              payloads.Firmware(fwType),
		VMType:              wl.VMType,
		InstancePersistence: payloads.Host,
		RequestedResources:  defaults,
		Networking:          networking,
	}

	if wl.VMType == payloads.Docker {
		startCmd.DockerImage = wl.ImageName
	}

	cmd := payloads.Start{
		Start: startCmd,
	}
	config.sc = cmd

	y, err := yaml.Marshal(&config.sc)
	if err != nil {
		glog.Warning("error marshalling config: ", err)
	}

	b, err := json.MarshalIndent(userData, "", "\t")
	if err != nil {
		glog.Warning("error marshalling user data: ", err)
	}

	config.config = "---\n" + string(y) + "...\n" + baseConfig + "---\n" + string(b) + "\n...\n"
	config.mac = networking.VnicMAC
	return
}

func newTenantHardwareAddr(ip net.IP) (hw net.HardwareAddr) {
	buf := make([]byte, 6)
	ipBytes := ip.To4()
	buf[0] |= 2
	buf[1] = 0
	copy(buf[2:6], ipBytes)
	hw = net.HardwareAddr(buf)
	return
}
