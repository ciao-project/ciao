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
	"time"

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/ciao-storage"
	"github.com/ciao-project/ciao/payloads"
	"github.com/golang/glog"
)

// CreateVolume will create a new block device and store it in the datastore.
func (c *controller) CreateVolume(tenant string, req api.RequestedVolume) (types.Volume, error) {
	var bd storage.BlockDevice

	var err error
	// no limits checking for now.
	if req.ImageRef != "" {
		// create bootable volume
		bd, err = c.CreateBlockDeviceFromSnapshot(req.ImageRef, "ciao-image")
		bd.Bootable = true
	} else if req.SourceVolID != "" {
		// copy existing volume
		bd, err = c.CopyBlockDevice(req.SourceVolID)
	} else {
		// create empty volume
		bd, err = c.CreateBlockDevice("", "", req.Size)
	}

	if err == nil && req.Size > bd.Size {
		bd.Size, err = c.Resize(bd.ID, req.Size)
	}

	if err != nil {
		return types.Volume{}, err
	}

	// store block device data in datastore
	// TBD - do we really need to do this, or can we associate
	// the block device data with the device itself?
	// you should modify BlockData to include a "bootable" flag.
	data := types.Volume{
		BlockDevice: bd,
		CreateTime:  time.Now(),
		TenantID:    tenant,
		State:       types.Available,
		Name:        req.Name,
		Description: req.Description,
		Internal:    req.Internal,
	}

	// It's best to make the quota request here as we don't know the volume
	// size earlier. If the ceph cluster is full then it might error out
	// earlier.
	resources := []payloads.RequestedResource{
		{Type: payloads.Volume, Value: 1},
		{Type: payloads.SharedDiskGiB, Value: bd.Size},
	}

	if !data.Internal {
		res := <-c.qs.Consume(tenant, resources...)

		if !res.Allowed() {
			_ = c.DeleteBlockDevice(bd.ID)
			c.qs.Release(tenant, res.Resources()...)
			return types.Volume{}, api.ErrQuota
		}
	}

	err = c.ds.AddBlockDevice(data)
	if err != nil {
		_ = c.DeleteBlockDevice(bd.ID)
		if !data.Internal {
			c.qs.Release(tenant, resources...)
		}
		return types.Volume{}, err
	}

	return data, nil
}

func (c *controller) DeleteVolume(tenant string, volume string) error {
	// get the block device information
	info, err := c.ds.GetBlockDevice(volume)
	if err != nil {
		return err
	}

	// check that the block device is owned by the tenant.
	if info.TenantID != tenant {
		return api.ErrVolumeOwner
	}

	// check that the block device is available.
	if info.State != types.Available {
		return api.ErrVolumeNotAvailable
	}

	// remove the block data from our datastore.
	err = c.ds.DeleteBlockDevice(volume)
	if err != nil {
		return err
	}

	// tell the underlying storage media to remove.
	err = c.DeleteBlockDevice(volume)
	if err != nil {
		return err
	}

	// release quota associated with this volume
	c.qs.Release(info.TenantID,
		payloads.RequestedResource{Type: payloads.Volume, Value: 1},
		payloads.RequestedResource{Type: payloads.SharedDiskGiB, Value: info.Size})

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
		return api.ErrVolumeNotAvailable
	}

	// check that the block device is owned by the tenant.
	if info.TenantID != tenant {
		return api.ErrVolumeOwner
	}

	// check that the instance is owned by the tenant.
	i, err := c.ds.GetTenantInstance(tenant, instance)
	if err != nil {
		return api.ErrInstanceNotFound
	}

	// update volume state to attaching
	info.State = types.Attaching

	err = c.ds.UpdateBlockDevice(info)
	if err != nil {
		return err
	}

	// create an attachment object
	a := payloads.StorageResource{
		ID:        info.ID,
		Ephemeral: false,
		Bootable:  false,
	}
	_, err = c.ds.CreateStorageAttachment(i.ID, a)
	if err != nil {
		info.State = types.Available
		dsErr := c.ds.UpdateBlockDevice(info)
		if dsErr != nil {
			glog.Error(dsErr)
		}
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
		return api.ErrVolumeNotAttached
	}

	// get the block device information
	info, err := c.ds.GetBlockDevice(volume)
	if err != nil {
		return err
	}

	// check that the block device is owned by the tenant.
	if info.TenantID != tenant {
		return api.ErrVolumeOwner
	}

	// check that the block device is in use
	if info.State != types.InUse {
		return api.ErrVolumeNotAttached
	}

	// we cannot detach a boot device - these aren't
	// like regular attachments and shouldn't be treated
	// as such.
	for _, a := range attachments {
		if a.Boot == true {
			return api.ErrVolumeNotAttached
		}
	}

	var retval error

	// detach everything for this volume
	for _, a := range attachments {
		// get instance info
		i, err := c.ds.GetTenantInstance(tenant, a.InstanceID)
		if err != nil {
			glog.Error(api.ErrInstanceNotFound)
			// keep going
			retval = err
			continue
		}

		i.StateLock.RLock()
		state := i.State
		i.StateLock.RUnlock()

		if state != payloads.Exited {
			retval = errors.New("Can only detach from exited instances")
			continue
		}

		// update volume state to detaching
		info.State = types.Available

		err = c.ds.UpdateBlockDevice(info)
		if err != nil {
			return err
		}
	}

	return retval
}

func (c *controller) ListVolumesDetail(tenant string) ([]types.Volume, error) {
	vols := []types.Volume{}

	devs, err := c.ds.GetBlockDevices(tenant)
	if err != nil {
		return vols, err
	}

	for _, vol := range devs {
		if vol.Internal {
			continue
		}

		vols = append(vols, vol)
	}

	return vols, nil
}

func (c *controller) ShowVolumeDetails(tenant string, volume string) (types.Volume, error) {
	vol, err := c.ds.GetBlockDevice(volume)
	if err != nil {
		return types.Volume{}, err
	}

	if vol.TenantID != tenant {
		return types.Volume{}, api.ErrVolumeOwner
	}

	return vol, nil
}
