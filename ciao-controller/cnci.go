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
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"net"
	"sync"
	"time"

	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/payloads"
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

var cnciEventTimeout = (2 * time.Minute)

// CNCI represents a cnci instance that manages a single subnet.
type CNCI struct {
	instance *types.Instance
	ctrl     *controller
	eventCh  *chan event
	subnet   string
	timer    *time.Timer
}

// CNCIManager is a structure which defines a manager for CNCI instances
// TBD: managing multiple CNCI instances.
type CNCIManager struct {
	tenant string
	ctrl   *controller

	// there's no reason to have separate lock for each map.
	cnciLock sync.RWMutex

	// this is a map of CNCI instance IDs to CNCI structs
	cncis map[string]*CNCI

	// this is a map of subnet strings to CNCI structs
	subnets map[string]*CNCI
}

func (c *CNCI) stop() error {
	err := c.instance.TransitionInstanceState(payloads.Stopping)
	if err != nil {
		return err
	}

	err = c.ctrl.deleteInstance(c.instance.ID)
	if err != nil {
		return errors.Wrapf(err, "error deleting CNCI instance")
	}

	return nil
}

func waitForEventTimeout(ch chan event, e event, timeout time.Duration) error {
	select {
	case recv := <-ch:
		if recv != e {
			return fmt.Errorf("expecting %v got %v", e, recv)
		}
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for event %v", e)
	}
}

func (c *CNCI) transitionState(to CNCIState) {
	glog.Infof("State transition to %s received for %s", to, c.instance.ID)

	err := c.instance.TransitionInstanceState(string(to))
	if err != nil {
		glog.Warningf("Error transitioning instance %s to %s state", c.instance.ID, string(to))
	}

	// some state changes cause events
	ch := c.eventCh

	if ch == nil {
		return
	}

	switch to {
	case exited:
		*ch <- removed
	case active:
		*ch <- added
	case failed:
		*ch <- startFailure
	}
}

func getTunnelIP(subnet string) net.IP {
	startTunnelIP := net.ParseIP(cnciNet.String())
	IP, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil
	}

	ones, bits := ipNet.Mask.Size()
	hostBits := bits - ones

	addr := binary.BigEndian.Uint32(IP.To4())
	mask := binary.BigEndian.Uint32(ipNet.Mask)
	start := binary.BigEndian.Uint32(startTunnelIP.To4())
	subnetNum := addr & mask

	// to calculate the tunnelIP, use the significant subnet
	// bits only. Since the top 12 bits are always the same,
	// get rid of them.
	tunnelNum := (subnetNum & 0x00cfffff) >> uint(hostBits)

	// add one to this value so that we don't allocate host 0
	tunnelNum++

	tunnelIP := make(net.IP, net.IPv4len)
	addr = start + uint32(tunnelNum)
	binary.BigEndian.PutUint32(tunnelIP, addr)

	return tunnelIP
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

	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	name := fmt.Sprintf("cnci-%s-%s", c.tenant, hex.EncodeToString(b))

	workloadID, err := c.ctrl.ds.GetCNCIWorkloadID()
	if err != nil {
		return nil, err
	}

	w := types.WorkloadRequest{
		WorkloadID: workloadID,
		TenantID:   c.tenant,
		Instances:  1,
		Subnet:     subnet,
		Name:       name,
	}

	instances, err := c.ctrl.startWorkload(w)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to Launch CNCI")
	}

	return instances[0], nil
}

// WaitForActive will launch a cnci if needed and wait for it to be active,
// or wait for an existing cnci to become active.
func (c *CNCIManager) WaitForActive(subnet string) error {
	c.cnciLock.Lock()

	cnci, ok := c.subnets[subnet]
	if ok {
		if cnci.timer != nil {
			cnci.timer.Stop()
			cnci.timer = nil
		}

		// subnet already exists
		c.cnciLock.Unlock()

		// block until subnet is active
		return c.waitForActive(subnet)
	}

	glog.V(2).Infof("cnci does not exist for subnet %s", subnet)

	ch := make(chan event)

	cnci = &CNCI{
		ctrl:    c.ctrl,
		eventCh: &ch,
		subnet:  subnet,
	}

	// we initialized the eventCh because we are going to wait for
	// an event. Close and delete at the conclusion of this function.
	defer func() {
		close(ch)
		cnci.eventCh = nil
	}()

	c.subnets[subnet] = cnci

	// send a launch command
	instance, err := c.launch(subnet)
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
	err = waitForEventTimeout(ch, added, cnciEventTimeout)
	if err != nil {
		return err
	}

	return c.refresh()
}

// ScheduleRemoveSubnet will kick off a timer to remove a subnet after 5 min.
// If a subnet is requested to be used again before the timer expires, the
// timer will get cancelled and the subnet will not be removed.
func (c *CNCIManager) ScheduleRemoveSubnet(subnet string) error {
	c.cnciLock.Lock()

	cnci, ok := c.subnets[subnet]
	if !ok {
		// there is no cnci for this subnet
		c.cnciLock.Unlock()
		return errors.New("Subnet doesn't exist")
	}

	if cnci.timer != nil {
		// we are already scheduled to remove
		c.cnciLock.Unlock()
		return nil
	}

	cnci.timer = time.AfterFunc(time.Minute*5, func() {
		c.cnciLock.Lock()
		cnci.timer = nil
		c.cnciLock.Unlock()

		err := c.RemoveSubnet(subnet)
		if err != nil {
			glog.Warningf("Unable to remove subnet: (%v)\n", err)
		}
	})

	c.cnciLock.Unlock()

	return nil
}

// RemoveSubnet is called when a subnet no longer is needed.
// a cnci can be stopped.
func (c *CNCIManager) RemoveSubnet(subnet string) error {
	glog.V(2).Infof("RemoveSubnet %s", subnet)

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

	ch := make(chan event)

	cnci.eventCh = &ch

	defer func() {
		close(ch)
		cnci.eventCh = nil
	}()

	c.cnciLock.Unlock()

	err = waitForEventTimeout(ch, removed, cnciEventTimeout)
	if err != nil {
		return err
	}

	return c.refresh()
}

// CNCIRemoved will move the CNCI back to the initial state
// and send an event through the event channel.
func (c *CNCIManager) CNCIRemoved(id string) error {
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

// CNCIStopped will move the CNCI to the exited state
// and send an event through the event channel.
func (c *CNCIManager) CNCIStopped(id string) error {
	c.cnciLock.Lock()
	defer c.cnciLock.Unlock()

	cnci, ok := c.cncis[id]
	if !ok {
		return errors.New("No CNCI found")
	}

	cnci.transitionState(exited)
	err := c.ctrl.restartInstance(cnci.instance.ID)

	return errors.Wrap(err, "Error restarting instance")
}

// CNCIAdded will move the CNCI into the active state
// and send an event through the event channel.
func (c *CNCIManager) CNCIAdded(id string) error {
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

	delete(c.cncis, id)
	delete(c.subnets, cnci.subnet)

	cnci.transitionState(failed)

	return nil
}

func (c *CNCIManager) waitForActive(subnet string) error {
	c.cnciLock.RLock()

	cnci, ok := c.subnets[subnet]

	c.cnciLock.RUnlock()

	if !ok {
		return errors.New("No CNCI found")
	}

	if instanceActive(cnci.instance) {
		return nil
	}

	// lock eventCh
	eCh := cnci.eventCh

	// CNCI launch not in process, and it's not active.
	if eCh == nil {
		return errors.New("CNCI not active")
	}

	// CNCI launch in process. we wait here till
	// the channel is closed. When it is, the cnci
	// is either active, or it failed to start.
	<-*eCh
	if instanceActive(cnci.instance) {
		return nil
	}

	return errors.New("CNCI not active")
}

func (c *CNCIManager) refresh() error {
	c.cnciLock.RLock()
	defer c.cnciLock.RUnlock()

	var cnciList []payloads.CNCINet

	// create a ConcentratorInstanceRefresh struct for each cnci
	for _, cnci := range c.cncis {
		tunnelID := crc32.ChecksumIEEE([]byte(c.tenant))
		tunnelIP := getTunnelIP(cnci.instance.Subnet)
		if tunnelIP == nil {
			return errors.New("Unable to derive CNCI tunnel IP")
		}

		r := payloads.CNCINet{
			PhysicalIP: cnci.instance.IPAddress,
			Subnet:     cnci.instance.Subnet,
			TunnelIP:   tunnelIP.String(),
			TunnelID:   tunnelID,
		}
		cnciList = append(cnciList, r)
	}

	// send the event to each cnci
	for _, cnci := range c.cncis {
		err := c.ctrl.client.CNCIRefresh(cnci.instance.ID, cnciList)
		if err != nil {
			// keep going, but log error.
			glog.Warningf("Unable to send cnci refresh to %s: (%v)", cnci.instance.ID, err)
		}
	}

	return nil
}

// GetInstanceCNCI will return the CNCI Instance for a specific tenant Instance
func (c *CNCIManager) GetInstanceCNCI(ID string) (*types.Instance, error) {
	// figure out what subnet we are looking for.
	instance, err := c.ctrl.ds.GetInstance(ID)
	if err != nil {
		return nil, err
	}

	c.cnciLock.Lock()
	defer c.cnciLock.Unlock()

	cnci, ok := c.subnets[instance.Subnet]
	if !ok {
		// there is no cnci for this subnet
		return nil, errors.New("Subnet doesn't exist")
	}

	return cnci.instance, nil
}

// GetSubnetCNCI will return the CNCI Instance for a specific subnet string
func (c *CNCIManager) GetSubnetCNCI(subnet string) (*types.Instance, error) {
	c.cnciLock.Lock()
	defer c.cnciLock.Unlock()

	cnci, ok := c.subnets[subnet]
	if !ok {
		// there is no cnci for this subnet
		return nil, errors.New("Subnet doesn't exist")
	}

	return cnci.instance, nil
}

func (c *CNCIManager) getInstanceCount(subnet string) (int, error) {
	var count int

	instances, err := c.ctrl.ds.GetAllInstancesFromTenant(c.tenant)
	if err != nil {
		return 0, err
	}

	for _, i := range instances {
		if i.Subnet == subnet {
			count++
		}
	}

	return count, nil
}

// Shutdown cleans up a CNCIManager in anticipation of a shutdown.
func (c *CNCIManager) Shutdown() {
	// the only thing we need to do right now at shutdown time
	// is to make sure any in progress timers are cancelled.
	c.cnciLock.Lock()
	defer c.cnciLock.Unlock()

	for _, cnci := range c.subnets {
		if cnci.timer != nil {
			cnci.timer.Stop()
			cnci.timer = nil
		}
	}
}

func newCNCIManager(ctrl *controller, tenant string) (*CNCIManager, error) {
	mgr := CNCIManager{
		tenant: tenant,
		ctrl:   ctrl,

		cncis:   make(map[string]*CNCI),
		subnets: make(map[string]*CNCI),
	}

	instances, err := ctrl.ds.GetTenantCNCIs(tenant)
	if err != nil {
		return nil, err
	}

	// you need to see if this cnci instance is actually needed
	// anymore.

	for _, i := range instances {
		cnci := CNCI{
			ctrl: ctrl,
		}

		cnci.instance = i

		cnci.subnet = i.Subnet
		mgr.cncis[i.ID] = &cnci
		mgr.subnets[i.Subnet] = &cnci

		// if we got shutdown prior to being able to remove
		// an unused subnet, we might be left with CNCIs that
		// are not needed. Check and see if any instances
		// in this subnet exist. If they don't, schedule this
		// cnci for removal after timeout.
		count, err := mgr.getInstanceCount(i.Subnet)
		if err != nil {
			return nil, err
		}

		if count == 0 {
			err = mgr.ScheduleRemoveSubnet(i.Subnet)
			if err != nil {
				// keep going, but log error.
				glog.Warningf("Unable to remove subnet (%v)", err)
			}
		}

	}

	return &mgr, nil
}

func shutdownCNCICtrls(c *controller) {
	// get all the current tenants
	ts, err := c.ds.GetAllTenants()
	if err != nil {
		return
	}

	// the only thing we need to do right now at shutdown time
	// is to make sure any in progress timers are cancelled.
	for _, t := range ts {
		t.CNCIctrl.Shutdown()
	}

	return
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
