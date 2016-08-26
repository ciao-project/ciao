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

package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/openstack/block"
	osIdentity "github.com/01org/ciao/openstack/identity"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

// Implement the Block Service interface
func (c *controller) GetAbsoluteLimits(tenant string) (block.AbsoluteLimits, error) {
	return block.AbsoluteLimits{}, nil
}

// CreateVolume will create a new block device and store it in the datastore.
// TBD: we need a better way to do bootable.
func (c *controller) CreateVolume(tenant string, req block.RequestedVolume) (block.Volume, error) {

	t, err := c.ds.GetTenant(tenant)
	if err != nil {
		return block.Volume{}, err
	}

	if t == nil {
		// go ahead and add this tenant
		if *noNetwork {
			_, err := c.ds.AddTenant(tenant)
			if err != nil {
				return block.Volume{}, err
			}
		} else {
			err = c.addTenant(tenant)
			if err != nil {
				return block.Volume{}, err
			}
		}
	}

	// no limits checking for now.
	bd, err := c.CreateBlockDevice(req.ImageRef, req.Size)
	if err != nil {
		return block.Volume{}, err
	}

	// store block device data in datastore
	// TBD - do we really need to do this, or can we associate
	// the block device data with the device itself?
	data := types.BlockData{
		BlockDevice: bd,
		Size:        req.Size,
		CreateTime:  time.Now(),
		TenantID:    tenant,
		State:       types.Available,
	}
	err = c.ds.AddBlockDevice(data)
	if err != nil {
		c.DeleteBlockDevice(bd.ID)
		return block.Volume{}, err
	}

	// convert our volume info into the openstack desired format.
	return block.Volume{
		Status:      block.Available,
		UserID:      tenant,
		Attachments: make([]block.Attachment, 0),
		Links:       make([]block.Link, 0),
		CreatedAt:   &data.CreateTime,
		ID:          bd.ID,
		Size:        data.Size,
		Bootable:    strconv.FormatBool(req.ImageRef != nil),
	}, nil
}

func (c *controller) DeleteVolume(tenant string, volume string) error {
	return nil
}

func (c *controller) AttachVolume(tenant string, volume string, instance string, mountpoint string) error {
	// get the block device information
	info, err := c.ds.GetBlockDevice(volume)
	if err != nil {
		return err
	}

	// check that the block device is available.
	if info.State != types.Available {
		return block.ErrVolumeNotAvailable
	}

	// check that the block device is owned by the tenant.
	if info.TenantID != tenant {
		return block.ErrVolumeOwner
	}

	// check that the instance is owned by the tenant.
	i, err := c.ds.GetInstance(instance)
	if err != nil {
		return block.ErrInstanceNotFound
	}

	if i.TenantID != tenant {
		return block.ErrInstanceOwner
	}

	if i.NodeID == "" {
		return block.ErrInstanceNotAvailable
	}

	// update volume state to attaching
	info.State = types.Attaching

	err = c.ds.UpdateBlockDevice(info)
	if err != nil {
		return err
	}

	// send command to attach volume.
	err = c.client.attachVolume(volume, instance, i.NodeID)
	if err != nil {
		info.State = types.Available
		dsErr := c.ds.UpdateBlockDevice(info)
		if dsErr != nil {
			glog.Error(dsErr)
		}
		return err
	}

	return nil
}

func (c *controller) DetachVolume(tenant string, volume string, attachment string) error {
	// we don't support detaching by attachment ID yet.
	if attachment != "" {
		return errors.New("Detaching by attachment ID not implemented")
	}

	// get attachment info
	attachments, err := c.ds.GetVolumeAttachments(volume)
	if err != nil {
		return err
	}

	if len(attachments) == 0 {
		return block.ErrVolumeNotAttached
	}

	// get the block device information
	info, err := c.ds.GetBlockDevice(volume)
	if err != nil {
		return err
	}

	// check that the block device is owned by the tenant.
	if info.TenantID != tenant {
		return block.ErrVolumeOwner
	}

	// check that the block device is in use
	if info.State != types.InUse {
		return block.ErrVolumeNotAttached
	}

	// update volume state to detaching
	info.State = types.Detaching

	err = c.ds.UpdateBlockDevice(info)
	if err != nil {
		return err
	}

	var retval error

	// detach everything for this volume
	for _, a := range attachments {
		// get instance info
		i, err := c.ds.GetInstance(a.InstanceID)
		if err != nil {
			glog.Error(block.ErrInstanceNotFound)
			// keep going
			retval = err
			continue
		}

		// send command to attach volume.
		err = c.client.detachVolume(a.BlockID, a.InstanceID, i.NodeID)
		if err != nil {
			retval = err
			glog.Errorf("Can't detach volume %s from instance %s\n", a.BlockID, a.InstanceID)
		}
	}

	return retval
}

func (c *controller) ListVolumes(tenant string) ([]block.ListVolume, error) {
	var vols []block.ListVolume

	data, err := c.ds.GetBlockDevices(tenant)
	if err != nil {
		return vols, err
	}

	for _, bd := range data {
		// TBD create links
		vol := block.ListVolume{
			ID: bd.ID,
		}
		vols = append(vols, vol)
	}

	return vols, nil
}

func (c *controller) ListVolumesDetail(tenant string) ([]block.VolumeDetail, error) {
	var vols []block.VolumeDetail

	devs, err := c.ds.GetBlockDevices(tenant)
	if err != nil {
		return vols, err
	}

	for _, data := range devs {
		var vol block.VolumeDetail

		vol.ID = data.ID
		vol.Size = data.Size
		vol.OSVolTenantAttr = data.TenantID
		vol.CreatedAt = &data.CreateTime

		switch data.State {
		case types.Attaching:
			vol.Status = block.Attaching
		case types.InUse:
			vol.Status = block.InUse
		case types.Available:
			vol.Status = block.Available
		default:
			vol.Status = block.VolumeStatus(data.State)
		}

		vols = append(vols, vol)
	}

	return vols, nil
}

func (c *controller) ShowVolumeDetails(tenant string, volume string) (block.VolumeDetail, error) {
	var vol block.VolumeDetail

	data, err := c.ds.GetBlockDevice(volume)
	if err != nil {
		return vol, err
	}

	if data.TenantID != tenant {
		return vol, block.ErrVolumeOwner
	}

	vol.ID = data.ID
	vol.Size = data.Size
	vol.OSVolTenantAttr = data.TenantID
	vol.CreatedAt = &data.CreateTime

	switch data.State {
	case types.Attaching:
		vol.Status = block.Attaching
	case types.InUse:
		vol.Status = block.InUse
	case types.Available:
		vol.Status = block.Available
	default:
		vol.Status = block.VolumeStatus(data.State)
	}

	return vol, nil
}

// Start will get the Volume API endpoints from the OpenStack block api,
// then wrap them in keystone validation. It will then start the https
// service.
func (c *controller) startVolumeService() error {
	config := block.APIConfig{Port: block.APIPort, VolService: c}

	r := block.Routes(config)
	if r == nil {
		return errors.New("Unable to start Volume Service")
	}

	// setup identity for these routes.
	validServices := []osIdentity.ValidService{
		{ServiceType: "volume", ServiceName: "ciao"},
		{ServiceType: "volumev2", ServiceName: "ciao"},
		{ServiceType: "volume", ServiceName: "cinder"},
		{ServiceType: "volumev2", ServiceName: "cinderv2"},
	}

	validAdmins := []osIdentity.ValidAdmin{
		{Project: "service", Role: "admin"},
		{Project: "admin", Role: "admin"},
	}

	err := r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		h := osIdentity.Handler{
			Client:        c.id.scV3,
			Next:          route.GetHandler(),
			ValidServices: validServices,
			ValidAdmins:   validAdmins,
		}

		route.Handler(h)

		return nil
	})

	if err != nil {
		return err
	}

	// start service.
	service := fmt.Sprintf(":%d", block.APIPort)

	return http.ListenAndServeTLS(service, httpsCAcert, httpsKey, r)
}
