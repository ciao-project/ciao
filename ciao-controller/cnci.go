// Copyright (c) 2017 Intel Corporation
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
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/payloads"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

// CNCIState represents specific allowed state strings
type CNCIState string

var (
	exited CNCIState = payloads.Exited
	active CNCIState = payloads.Running
	failed CNCIState = payloads.ExitFailed
)

type event string

var (
	added        event = "concentrator added"
	startFailure event = "cnci start failure"
	removed      event = "concentrator removed"
)

var cnciEventTimeout = (5 * time.Minute)

// CNCI represents a cnci instance that manages a single subnet.
type CNCI struct {
	instance *types.Instance
	ctrl     *controller
	eventCh  chan event
	subnet   int
}

// CNCIManager is a structure which defines a manager for CNCI instances
// TBD: managing multiple CNCI instances.
type CNCIManager struct {
	tenant string
	ctrl   *controller

	// there's no reason to have separate lock for each map.
	cnciLock *sync.RWMutex
	cncis    map[string]*CNCI
	subnets  map[int]*CNCI
}

func (c *CNCI) stop() error {
	err := transitionInstanceState(c.instance, payloads.Stopping)
	if err != nil {
		return err
	}

	err = c.ctrl.deleteInstance(c.instance.ID)
	if err != nil {
		return errors.Wrapf(err, "error deleting CNCI instance")
	}

	return nil
}

func (c *CNCI) waitForEventTimeout(e event, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case recv := <-c.eventCh:
		if recv != e {
			return fmt.Errorf("expecting %v got %v", e, recv)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for event %v", e)
	}
}

func (c *CNCI) transitionState(to CNCIState) {
	transitionInstanceState(c.instance, (string(to)))

	// some state changes cause events
	switch to {
	case exited:
		c.eventCh <- removed
	case active:
		c.eventCh <- added
	case failed:
		c.eventCh <- startFailure
	}
}

// Active will return true if the CNCI has been launched successfully
func (c *CNCIManager) Active(ID string) bool {
	c.cnciLock.RLock()
	defer c.cnciLock.RUnlock()

	cnci, ok := c.cncis[ID]
	if !ok {
		return false
	}

	return instanceActive(cnci.instance)
}

func (c *CNCIManager) launch(subnet string) (*types.Instance, error) {
	glog.V(2).Infof("launching cnci for subnet %s", subnet)

	workloadID, err := c.ctrl.ds.GetCNCIWorkloadID()
	if err != nil {
		return nil, err
	}

	w := types.WorkloadRequest{
		WorkloadID: workloadID,
		TenantID:   c.tenant,
		Instances:  1,
		Subnet:     subnet,
		Name:       fmt.Sprintf("CNCI for subnet %s", subnet),
	}

	instances, err := c.ctrl.startWorkload(w)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to Launch CNCI")
	}

	return instances[0], nil
}

// WaitForActive will launch a cnci if needed and wait for it to be active,
// or wait for an existing cnci to become active.
func (c *CNCIManager) WaitForActive(subnet int) error {
	c.cnciLock.Lock()

	cnci, ok := c.subnets[subnet]
	if ok {
		// subnet already exists
		c.cnciLock.Unlock()

		// block until subnet is active
		return c.waitForActive(subnet)
	}

	cnci = &CNCI{
		ctrl:    c.ctrl,
		eventCh: make(chan event),
		subnet:  subnet,
	}

	c.subnets[subnet] = cnci

	subnetBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(subnetBytes, uint16(subnet))
	ip := net.IPv4(172, subnetBytes[0], subnetBytes[1], 0)
	ipNet := net.IPNet{
		IP:   ip,
		Mask: net.IPv4Mask(255, 255, 255, 0),
	}
	subnetStr := ipNet.String()

	// send a launch command
	instance, err := c.launch(subnetStr)
	if err != nil {
		c.cnciLock.Unlock()
		return err
	}

	glog.V(2).Infof("AddSubnet CNCI instance is %s", instance.ID)

	cnci.instance = instance
	cnci.subnet = subnet

	c.cncis[instance.ID] = cnci

	c.cnciLock.Unlock()

	// we release the lock before waiting because
	// we need to be able to read the event channel.
	return cnci.waitForEventTimeout(added, cnciEventTimeout)
}

// RemoveSubnet is called when a subnet no longer is needed.
// a cnci can be stopped.
func (c *CNCIManager) RemoveSubnet(subnet int) error {
	c.cnciLock.Lock()

	cnci, ok := c.subnets[subnet]
	if !ok {
		// there is no cnci for this subnet
		c.cnciLock.Unlock()
		return errors.New("Subnet doesn't exist")
	}

	delete(c.subnets, subnet)

	err := cnci.stop()
	if err != nil {
		c.cnciLock.Unlock()
		return err
	}

	c.cnciLock.Unlock()

	err = cnci.waitForEventTimeout(removed, cnciEventTimeout)
	if err != nil {
		return err
	}

	return nil
}

// ConcentratorInstanceRemoved will move the CNCI back to the initial state
// and send an event through the event channel.
func (c *CNCIManager) ConcentratorInstanceRemoved(id string) error {
	c.cnciLock.Lock()
	defer c.cnciLock.Unlock()

	cnci, ok := c.cncis[id]
	if !ok {
		return errors.New("No CNCI found")
	}

	cnci.transitionState(exited)

	delete(c.cncis, cnci.instance.ID)

	return nil
}

// ConcentratorInstanceAdded will move the CNCI into the active state
// and send an event through the event channel.
func (c *CNCIManager) ConcentratorInstanceAdded(id string) error {
	c.cnciLock.Lock()
	defer c.cnciLock.Unlock()

	cnci, ok := c.cncis[id]
	if !ok {
		return errors.New("No CNCI found")
	}

	cnci.transitionState(active)

	return nil
}

// StartFailure will move the CNCI to the error state and
// send an event through the event channel.
func (c *CNCIManager) StartFailure(id string) error {
	c.cnciLock.Lock()
	defer c.cnciLock.Unlock()

	cnci, ok := c.cncis[id]
	if !ok {
		return errors.New("No CNCI found")
	}

	cnci.transitionState(failed)

	// we should probably not do this, and instead we should
	// delete from the map?
	cnci.instance = nil

	return nil
}

func (c *CNCIManager) waitForActive(subnet int) error {
	c.cnciLock.RLock()

	cnci, ok := c.subnets[subnet]

	c.cnciLock.RUnlock()

	if !ok {
		return errors.New("No CNCI found")
	}

	if instanceActive(cnci.instance) {
		return nil
	}

	ch := make(chan bool)

	// poll with timeout

	ticker := time.NewTicker(time.Millisecond * 500)

	go func() {
		for range ticker.C {
			if instanceActive(cnci.instance) {
				ch <- true
				close(ch)
				return
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), cnciEventTimeout)
	defer cancel()

	select {
	case <-ch:
		ticker.Stop()
		return nil
	case <-ctx.Done():
		ticker.Stop()
		return errors.New("timeout waiting for CNCI active")
	}
}

// GetInstanceCNCI will return the CNCI Instance for a specific tenant Instance
func (c *CNCIManager) GetInstanceCNCI(ID string) (*types.Instance, error) {
	// figure out what subnet we are looking for.
	instance, err := c.ctrl.ds.GetInstance(ID)
	if err != nil {
		return nil, err
	}

	// convert subnet string to int
	ipAddr := net.ParseIP(instance.IPAddress)
	if ipAddr == nil {
		return nil, errors.New("Invalid instance IP address")
	}

	ipBytes := ipAddr.To4()
	if ipBytes == nil {
		return nil, errors.New("Unable to convert ip to bytes")
	}

	subnetInt := binary.BigEndian.Uint16(ipBytes[1:3])

	c.cnciLock.Lock()
	defer c.cnciLock.Unlock()

	cnci, ok := c.subnets[int(subnetInt)]
	if !ok {
		// there is no cnci for this subnet
		return nil, errors.New("Subnet doesn't exist")
	}

	return cnci.instance, nil
}

func subnetStringToInt(cidr string) (int, error) {
	ipAddr, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return 0, err
	}

	ipBytes := ipAddr.To4()
	if ipBytes == nil {
		return 0, errors.New("Unable to convert ip to bytes")
	}

	return int(binary.BigEndian.Uint16(ipBytes[1:3])), nil
}

// GetSubnetCNCI will return the CNCI Instance for a specific subnet string
func (c *CNCIManager) GetSubnetCNCI(subnet string) (*types.Instance, error) {
	subnetInt, err := subnetStringToInt(subnet)
	if err != nil {
		return nil, err
	}

	c.cnciLock.Lock()
	defer c.cnciLock.Unlock()

	cnci, ok := c.subnets[subnetInt]
	if !ok {
		// there is no cnci for this subnet
		return nil, errors.New("Subnet doesn't exist")
	}

	return cnci.instance, nil
}

func newCNCIManager(ctrl *controller, tenant string) (*CNCIManager, error) {
	mgr := CNCIManager{
		tenant: tenant,
		ctrl:   ctrl,

		cnciLock: &sync.RWMutex{},
		cncis:    make(map[string]*CNCI),
		subnets:  make(map[int]*CNCI),
	}

	instances, err := ctrl.ds.GetTenantCNCIs(tenant)
	if err != nil {
		return nil, err
	}

	for _, i := range instances {
		cnci := CNCI{
			ctrl:    ctrl,
			eventCh: make(chan event),
		}

		cnci.instance = i

		// convert cnci instance string to int for map
		subnetInt, err := subnetStringToInt(i.Subnet)
		if err != nil {
			return nil, err
		}

		cnci.subnet = subnetInt

		mgr.cncis[i.ID] = &cnci

		mgr.subnets[subnetInt] = &cnci
	}

	return &mgr, nil
}

func initializeCNCICtrls(c *controller) error {
	// get all the current tenants
	ts, err := c.ds.GetAllTenants()
	if err != nil {
		return errors.Wrap(err, "error getting tenants")
	}

	for _, t := range ts {
		t.CNCIctrl, err = newCNCIManager(c, t.ID)
		if err != nil {
			return errors.Wrap(err, "error allocating CNCI manager")
		}
	}

	return nil
}
