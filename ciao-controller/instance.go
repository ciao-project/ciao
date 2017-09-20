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
	"net"
	"sync"
	"time"

	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/ciao-controller/utils"
	"github.com/ciao-project/ciao/ciao-storage"
	"github.com/ciao-project/ciao/payloads"
	"github.com/ciao-project/ciao/ssntp/uuid"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
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
	ctl       *controller
	startTime time.Time
}

type userData struct {
	UUID     string `json:"uuid"`
	Hostname string `json:"hostname"`
}

func isCNCIWorkload(workload *types.Workload) bool {
	for r := range workload.Defaults {
		if workload.Defaults[r].Type == payloads.NetworkNode {
			return true
		}
	}

	return false
}

func newInstance(ctl *controller, tenantID string, workload *types.Workload,
	volumes []storage.BlockDevice, name string, subnet string) (*instance, error) {
	id := uuid.Generate()

	if name != "" {
		existingID, err := ctl.ds.ResolveInstance(tenantID, name)
		if err != nil {
			return nil, errors.Wrap(err, "error trying to resolve name")
		}

		if existingID != "" {
			return nil, fmt.Errorf("Instance name already in use: %s", name)
		}
	}

	config, err := newConfig(ctl, workload, id.String(), tenantID, volumes, name)
	if err != nil {
		return nil, err
	}

	newInstance := types.Instance{
		TenantID:    tenantID,
		WorkloadID:  workload.ID,
		State:       payloads.Pending,
		ID:          id.String(),
		CNCI:        config.cnci,
		IPAddress:   config.ip,
		VnicUUID:    config.sc.Start.Networking.VnicUUID,
		Subnet:      config.sc.Start.Networking.Subnet,
		MACAddress:  config.mac,
		CreateTime:  time.Now(),
		Name:        name,
		StateLock:   &sync.RWMutex{},
		StateChange: sync.NewCond(&sync.Mutex{}),
	}

	if subnet != "" {
		newInstance.Subnet = subnet
	}

	i := &instance{
		ctl:       ctl,
		newConfig: config,
		Instance:  newInstance,
	}

	return i, nil
}

func (i *instance) Add() error {
	ds := i.ctl.ds
	var err error
	err = ds.AddInstance(&i.Instance)
	if err != nil {
		return errors.Wrapf(err, "Error creating instance in datastore")
	}

	for _, volume := range i.newConfig.sc.Start.Storage {
		if volume.ID == "" && volume.Local {
			// these are launcher auto-created ephemeral
			continue
		}
		_, err = ds.GetBlockDevice(volume.ID)
		if err != nil {
			return fmt.Errorf("Invalid block device mapping.  %s already in use", volume.ID)
		}

		_, err = ds.CreateStorageAttachment(i.Instance.ID, volume)
		if err != nil {
			return errors.Wrap(err, "Error creating storage attachment")
		}
	}

	return nil
}

func (i *instance) Clean() error {
	if i.CNCI {
		// CNCI resources are not tracked by quota system
		return nil
	}

	i.ctl.ds.ReleaseTenantIP(i.TenantID, i.IPAddress)

	wl, err := i.ctl.ds.GetWorkload(i.TenantID, i.WorkloadID)
	if err != nil {
		return errors.Wrap(err, "error getting workload from datastore")
	}
	resources := []payloads.RequestedResource{{Type: payloads.Instance, Value: 1}}
	resources = append(resources, wl.Defaults...)
	i.ctl.qs.Release(i.TenantID, resources...)
	i.ctl.deleteEphemeralStorage(i.ID)
	return nil
}

func (i *instance) Allowed() (bool, error) {
	if i.CNCI == true {
		// should I bother to check the tenant id exists?
		return true, nil
	}

	ds := i.ctl.ds

	wl, err := ds.GetWorkload(i.TenantID, i.WorkloadID)
	if err != nil {
		return true, errors.Wrap(err, "error getting workload from datastore")
	}

	resources := []payloads.RequestedResource{{Type: payloads.Instance, Value: 1}}
	resources = append(resources, wl.Defaults...)
	res := <-i.ctl.qs.Consume(i.TenantID, resources...)

	// Cleanup on disallowed happens in Clean()
	return res.Allowed(), nil
}

func transitionInstanceState(i *types.Instance, to string) error {
	i.StateLock.Lock()
	defer i.StateLock.Unlock()

	glog.V(2).Infof("Instance %s: %s -> %s", i.ID, i.State, to)

	switch to {
	case payloads.Stopping:
		if i.State != payloads.Running {
			return errors.New("Stop operation not allowed")
		}
	case payloads.Running:
		if i.State != payloads.Pending {
			return errors.New("Set active without pending")
		}
	}

	i.StateChange.L.Lock()
	i.State = to
	i.StateChange.L.Unlock()
	i.StateChange.Signal()

	return nil
}

func instanceActive(i *types.Instance) bool {
	i.StateLock.RLock()
	defer i.StateLock.RUnlock()

	if i.State == payloads.Running {
		return true
	}

	return false
}

func addBlockDevice(c *controller, tenant string, instanceID string, device storage.BlockDevice, s types.StorageResource) (payloads.StorageResource, error) {
	// don't you need to add support for indicating whether
	// a block device is bootable.
	data := types.BlockData{
		BlockDevice: device,
		CreateTime:  time.Now(),
		TenantID:    tenant,
		Name:        fmt.Sprintf("Storage for instance: %s", instanceID),
		Description: s.Tag,
		Internal:    s.Internal,
	}

	if !data.Internal {
		res := <-c.qs.Consume(tenant,
			payloads.RequestedResource{Type: payloads.Volume, Value: 1},
			payloads.RequestedResource{Type: payloads.SharedDiskGiB, Value: device.Size})

		if !res.Allowed() {
			c.DeleteBlockDevice(device.ID)
			c.qs.Release(tenant, res.Resources()...)
			return payloads.StorageResource{}, fmt.Errorf("Error creating volume: %s", res.Reason())
		}
	}

	err := c.ds.AddBlockDevice(data)
	if err != nil {
		c.DeleteBlockDevice(device.ID)
		return payloads.StorageResource{}, err
	}

	return payloads.StorageResource{ID: data.ID, Bootable: s.Bootable, Ephemeral: s.Ephemeral}, nil
}

func getStorage(c *controller, s types.StorageResource, tenant string, instanceID string) (payloads.StorageResource, error) {
	// storage already exists, use preexisting definition.
	if s.ID != "" {
		return payloads.StorageResource{ID: s.ID, Bootable: s.Bootable}, nil
	}

	// new storage.
	// TBD: handle all these cases
	// - create bootable volume from image.
	//   assumptions: SourceType is "image"
	//                Bootable is true
	//                SourceID points to existing image
	// - create bootable volume from volume.
	//   Assumptions: SourceType is "volume"
	//                Bootable is true
	//                SourceID points to existing volume
	// - create attachable empty volume.
	//   Assumptions: SourceType is "empty"
	//                Bootable is ignored
	//                SourceID is ignored
	// - create attachable volume from image?
	//   Assumptions: SourceType is "image"
	//                Bootable is false
	//                SourceID points to existing image
	// - create attachable volume from volume.
	//   Assumptions: SourceType is "volume"
	//                Bootable is false
	//                SourceID points to existing volume.
	// assume always persistent for now.
	// assume we have already checked quotas.
	// ID of source is the image id.
	var device storage.BlockDevice
	var err error
	switch s.SourceType {
	case types.ImageService:
		device, err = c.CreateBlockDeviceFromSnapshot(s.SourceID, "ciao-image")
		if err != nil {
			glog.Errorf("Unable to get block device for image: %v", err)
			return payloads.StorageResource{}, err
		}

	case types.VolumeService:
		device, err = c.CopyBlockDevice(s.SourceID)
		if err != nil {
			return payloads.StorageResource{}, err
		}

	case types.Empty:
		device, err = c.CreateBlockDevice("", "", s.Size)
		if err != nil {
			return payloads.StorageResource{}, err
		}

	default:
		return payloads.StorageResource{}, errors.New("Unsupported workload storage variant in getStorage()")
	}

	if device.Size < s.Size {
		device.Size, err = c.Resize(device.ID, s.Size)
	}

	if err != nil {
		c.DeleteBlockDevice(device.ID)
		return payloads.StorageResource{}, errors.Wrap(err, "error resizing volume")
	}

	return addBlockDevice(c, tenant, instanceID, device, s)
}

func controllerStorageResourceFromPayload(volume payloads.StorageResource) (s types.StorageResource) {
	s.ID = volume.ID
	s.Bootable = volume.Bootable
	s.Ephemeral = volume.Ephemeral
	s.Size = volume.Size
	s.SourceType = ""
	s.SourceID = ""
	s.Tag = volume.Tag

	return
}

func networkConfig(ctl *controller, tenant *types.Tenant, networking *payloads.NetworkResources, cnci bool) error {
	networking.VnicUUID = uuid.Generate().String()

	if cnci {
		hwaddr, err := utils.NewHardwareAddr()
		if err != nil {
			return err
		}

		networking.VnicMAC = hwaddr.String()
		return nil
	}

	ipAddress, err := ctl.ds.AllocateTenantIP(tenant.ID)
	if err != nil {
		fmt.Println("Unable to allocate IP address: ", err)
		return err
	}

	networking.VnicMAC = utils.NewTenantHardwareAddr(ipAddress).String()

	// send in CIDR notation?
	networking.PrivateIP = ipAddress.String()
	mask := net.IPv4Mask(255, 255, 255, 0)
	ipnet := net.IPNet{
		IP:   ipAddress.Mask(mask),
		Mask: mask,
	}
	networking.Subnet = ipnet.String()

	cnciInstance, err := tenant.CNCIctrl.GetSubnetCNCI(networking.Subnet)
	if err != nil {
		return err
	}

	networking.ConcentratorUUID = cnciInstance.ID

	// in theory we should refuse to go on if ip is null
	// for now let's keep going
	networking.ConcentratorIP = cnciInstance.IPAddress
	return nil
}

func storageConfig(ctl *controller, tenant *types.Tenant, instanceID string, volumes []storage.BlockDevice) ([]payloads.StorageResource, error) {
	var storage []payloads.StorageResource

	// handle storage resources for just this instance
	for _, volume := range volumes {
		instanceStorage := payloads.StorageResource{
			ID:        volume.ID,
			Bootable:  volume.Bootable,
			Ephemeral: volume.Ephemeral,
			Local:     volume.Local,
			Swap:      volume.Swap,
			BootIndex: volume.BootIndex,
			Tag:       volume.Tag,
			Size:      volume.Size,
		}

		// controller created (as opposed to launcher
		// created) instance storage (workload storage is later)
		if volume.ID == "" && !volume.Local {
			// auto-create empty
			device, err := ctl.CreateBlockDevice("", "", volume.Size)
			if err != nil {
				return storage, err
			}

			instanceStorage.ID = device.ID
			s := controllerStorageResourceFromPayload(instanceStorage)
			_, err = addBlockDevice(ctl, tenant.ID, instanceID, device, s)
			if err != nil {
				return storage, err
			}
		} /* else {
			// volume.ID != "": launcher will attach pre-existing volume
			// volume.Local: launcher will create ephemeral volume
		} */

		storage = append(storage, instanceStorage)
	}

	return storage, nil
}

func newConfig(ctl *controller, wl *types.Workload, instanceID string, tenantID string,
	volumes []storage.BlockDevice, name string) (config, error) {
	var metaData userData
	var config config
	var networking payloads.NetworkResources
	var storage []payloads.StorageResource

	baseConfig := wl.Config
	defaults := wl.Defaults
	fwType := wl.FWType
	config.cnci = isCNCIWorkload(wl)
	metaData.UUID = instanceID

	tenant, err := ctl.ds.GetTenant(tenantID)
	if err != nil {
		fmt.Println("unable to get tenant")
	}

	err = networkConfig(ctl, tenant, &networking, config.cnci)
	if err != nil {
		return config, err
	}

	metaData.Hostname = instanceID
	if name != "" {
		metaData.Hostname = name
	}

	storage, err = storageConfig(ctl, tenant, instanceID, volumes)
	if err != nil {
		return config, err
	}

	config.ip = networking.PrivateIP

	// handle storage resources in workload definition
	for i := range wl.Storage {
		workloadStorage, err := getStorage(ctl, wl.Storage[i], tenantID, instanceID)
		if err != nil {
			return config, err
		}
		storage = append(storage, workloadStorage)
	}

	// hardcode persistence until changes can be made to workload
	// template datastore.  Estimated resources can be blank
	// for now because we don't support it yet.
	startCmd := payloads.StartCmd{
		TenantUUID:          tenantID,
		InstanceUUID:        instanceID,
		FWType:              payloads.Firmware(fwType),
		VMType:              wl.VMType,
		InstancePersistence: payloads.Host,
		RequestedResources:  defaults,
		Networking:          networking,
		Storage:             storage,
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

	b, err := json.MarshalIndent(metaData, "", "\t")
	if err != nil {
		glog.Warning("error marshalling user data: ", err)
	}

	config.config = "---\n" + string(y) + "...\n" + baseConfig + "---\n" + string(b) + "\n...\n"
	config.mac = networking.VnicMAC

	return config, err
}
