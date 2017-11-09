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
	"fmt"
	"net"
	"runtime"
	"time"

	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/payloads"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

func (c *controller) restartInstance(instanceID string) error {
	// should I bother to see if instanceID is valid?
	i, err := c.ds.GetInstance(instanceID)
	if err != nil {
		return err
	}

	if i.State != "exited" {
		return errors.New("You may only restart paused instances")
	}

	w, err := c.ds.GetWorkload(i.WorkloadID)
	if err != nil {
		return err
	}

	t, err := c.ds.GetTenant(i.TenantID)
	if err != nil {
		return err
	}

	if !i.CNCI {
		err = t.CNCIctrl.WaitForActive(i.Subnet)
		if err != nil {
			return errors.Wrap(err, "Error waiting for active subnet")
		}
	}

	go func() {
		if err := c.client.RestartInstance(i, &w, t); err != nil {
			glog.Warningf("Error restarting instance: %v", err)
		}
	}()

	return nil
}

func (c *controller) stopInstance(instanceID string) error {
	// get node id.  If there is no node id we can't send a delete
	i, err := c.ds.GetInstance(instanceID)
	if err != nil {
		return err
	}

	if i.NodeID == "" {
		return types.ErrInstanceNotAssigned
	}

	if i.State == payloads.ComputeStatusPending {
		return errors.New("You may not stop a pending instance")
	}

	go func() {
		if err := c.client.StopInstance(instanceID, i.NodeID); err != nil {
			glog.Warningf("Error stopping instance: %v", err)
		}
	}()

	return nil
}

// delete an instance, wait for the deleted event.
func (c *controller) deleteInstanceSync(instanceID string) error {
	wait := make(chan struct{})

	i, err := c.ds.GetInstance(instanceID)
	if err != nil {
		return err
	}

	err = c.deleteInstance(instanceID)
	if err != nil {
		return err
	}

	go func() {
		i.StateChange.L.Lock()
		for {
			i.StateLock.RLock()
			if i.State == payloads.Deleted || i.State == payloads.Hung {
				break
			}
			glog.V(2).Infof("waiting for %s to be deleted", i.ID)
			i.StateLock.RUnlock()
			i.StateChange.Wait()
		}

		i.StateLock.RUnlock()
		i.StateChange.L.Unlock()

		glog.V(2).Infof("%s is hung or deleted", i.ID)
		close(wait)
	}()

	select {
	case <-wait:
		return nil
	case <-time.After(2 * time.Minute):
		err = transitionInstanceState(i, payloads.Hung)
		if err != nil {
			glog.Warningf("Error transitioning instance to hung state: %v", err)
		}
		return fmt.Errorf("timeout waiting for delete")
	}
}

func (c *controller) deleteInstance(instanceID string) error {
	// get node id.  If there is no node id and the instance is
	// pending we can't send a delete
	i, err := c.ds.GetInstance(instanceID)
	if err != nil {
		return err
	}

	if i.NodeID == "" && i.State == payloads.Pending {
		return types.ErrInstanceNotAssigned
	}

	// check for any external IPs
	IPs := c.ds.GetMappedIPs(&i.TenantID)
	for _, m := range IPs {
		if m.InstanceID == instanceID {
			return types.ErrInstanceMapped
		}
	}

	go func() {
		if err := c.client.DeleteInstance(instanceID, i.NodeID); err != nil {
			glog.Warningf("Error deleting instance: %v", err)
		}
	}()

	return nil
}

func (c *controller) confirmTenantRaw(tenantID string) error {
	tenant, err := c.ds.GetTenant(tenantID)
	if err != nil {
		return err
	}

	if tenant != nil {
		return nil
	}

	// if we are adding tenant this way, we need to use defaults
	config := types.TenantConfig{
		Name:       "",
		SubnetBits: 24,
	}

	tenant, err = c.ds.AddTenant(tenantID, config)
	if err != nil {
		return err
	}

	tenant.CNCIctrl, err = newCNCIManager(c, tenantID)
	if err != nil {
		return err
	}

	return nil
}

func (c *controller) confirmTenant(tenantID string) error {
	c.tenantReadinessLock.Lock()
	memo := c.tenantReadiness[tenantID]
	if memo != nil {

		// Someone else has already or is in the process of confirming
		// this tenant.  We need to wait until memo.ch is closed before
		// continuing.

		c.tenantReadinessLock.Unlock()
		<-memo.ch
		if memo.err != nil {
			return memo.err
		}

		// If we get here we know that confirmTenantRaw has already
		// been successfully called for this tenant during the life
		// time of this controller invocation.

		return nil
	}

	ch := make(chan struct{})
	c.tenantReadiness[tenantID] = &tenantConfirmMemo{ch: ch}
	c.tenantReadinessLock.Unlock()
	err := c.confirmTenantRaw(tenantID)
	if err != nil {
		c.tenantReadinessLock.Lock()
		c.tenantReadiness[tenantID].err = err
		delete(c.tenantReadiness, tenantID)
		c.tenantReadinessLock.Unlock()
	}
	close(ch)
	return err
}

func (c *controller) createInstance(w types.WorkloadRequest, wl types.Workload, name string, newIP net.IP) (*types.Instance, error) {
	startTime := time.Now()

	instance, err := newInstance(c, w.TenantID, &wl, w.Volumes, name, w.Subnet, newIP)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating instance")
	}
	instance.startTime = startTime

	ok, err := instance.Allowed()
	if err != nil {
		_ = instance.Clean()
		return nil, errors.Wrap(err, "Error checking if instance allowed")
	}

	if !ok {
		_ = instance.Clean()
		return nil, errors.New("Over quota")
	}

	err = instance.Add()
	if err != nil {
		_ = instance.Clean()
		return nil, errors.Wrap(err, "Error adding instance")
	}

	if w.TraceLabel == "" {
		err = c.client.StartWorkload(instance.newConfig.config)
	} else {
		err = c.client.StartTracedWorkload(instance.newConfig.config, instance.startTime, w.TraceLabel)
	}

	if err != nil {
		_ = instance.Clean()
		return nil, errors.Wrap(err, "Error starting workload")
	}

	return instance.Instance, nil
}

func (c *controller) startWorkload(w types.WorkloadRequest) ([]*types.Instance, error) {
	var e error
	var sem = make(chan int, runtime.NumCPU())

	if w.Instances <= 0 {
		return nil, errors.New("Missing number of instances to start")
	}

	wl, err := c.ds.GetWorkload(w.WorkloadID)
	if err != nil {
		return nil, err
	}

	var IPPool []net.IP

	// if this is for a CNCI, we don't want to allocate any IPs.
	if w.Subnet == "" {
		IPPool, err = c.ds.AllocateTenantIPPool(w.TenantID, w.Instances)
		if err != nil {
			return nil, err
		}
	}

	var newInstances []*types.Instance
	type result struct {
		instance *types.Instance
		err      error
	}

	errChan := make(chan result)

	for i := 0; i < w.Instances; i++ {
		var newIP net.IP

		if w.Subnet == "" {
			newIP = IPPool[i]
		}

		name := w.Name
		if name != "" {
			if w.Instances > 1 {
				name = fmt.Sprintf("%s-%d", name, i)
			}
		}

		go func(newIP net.IP, name string) {
			sem <- 1
			var err error
			var instance *types.Instance
			defer func() {
				ret := result{
					err:      err,
					instance: instance,
				}
				<-sem
				errChan <- ret
			}()

			instance, err = c.createInstance(w, wl, name, newIP)
			if err != nil {
				err = errors.Wrap(err, "Error creating instance")
				return
			}
		}(newIP, name)
	}

	for i := 0; i < w.Instances; i++ {
		retVal := <-errChan
		if e == nil {
			// return the first error
			e = retVal.err
		}
		newInstances = append(newInstances, retVal.instance)
	}

	return newInstances, e
}

func (c *controller) deleteEphemeralStorage(instanceID string) error {
	attachments := c.ds.GetStorageAttachments(instanceID)
	for _, attachment := range attachments {
		if !attachment.Ephemeral {
			continue
		}
		err := c.ds.DeleteStorageAttachment(attachment.ID)
		if err != nil {
			return errors.Wrap(err, "Error deleting storage attachment from datastore")
		}
		bd, err := c.ds.GetBlockDevice(attachment.BlockID)
		if err != nil {
			return errors.Wrap(err, "Error getting block device from datastore")
		}
		err = c.ds.DeleteBlockDevice(attachment.BlockID)
		if err != nil {
			return errors.Wrap(err, "Error deleting block device from datastore")
		}
		err = c.DeleteBlockDevice(attachment.BlockID)
		if err != nil {
			return errors.Wrap(err, "Error deleting block device")
		}
		if !bd.Internal {
			c.qs.Release(bd.TenantID,
				payloads.RequestedResource{Type: payloads.Volume, Value: 1},
				payloads.RequestedResource{Type: payloads.SharedDiskGiB, Value: bd.Size})
		}
	}
	return nil
}
