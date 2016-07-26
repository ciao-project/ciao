package main

import (
	"time"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/openstack/block"
	"github.com/golang/glog"
)

// Implement the Block Service interface
func (c *controller) GetAbsoluteLimits(tenant string) (block.AbsoluteLimits, error) {
	return block.AbsoluteLimits{}, nil
}

func (c *controller) CreateVolume(tenant string, req block.RequestedVolume) (block.Volume, error) {
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
	return make([]block.VolumeDetail, 0), nil
}

func (c *controller) ShowVolumeDetails(tenant string) (block.VolumeDetail, error) {
	return block.VolumeDetail{}, nil
}
