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

// Package datastore retrieves stores data for the ciao controller.
// This package caches most data in memory, and uses a sql
// database as persistent storage.
package datastore

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/payloads"
	"github.com/golang/glog"
	"net"
	"sort"
	"sync"
	"time"
)

type Config struct {
	PersistentURI     string
	TransientURI      string
	InitTablesPath    string
	InitWorkloadsPath string
}

type userEventType string

const (
	userInfo  userEventType = "info"
	userWarn  userEventType = "warn"
	userError userEventType = "error"
)

type workload struct {
	types.Workload
	filename string
}

type tenant struct {
	types.Tenant
	network   map[int]map[int]bool
	subnets   []int
	instances map[string]*types.Instance
}

type node struct {
	types.Node
	instances map[string]*types.Instance
}

type persistentStore interface {
	disconnect()

	// interfaces related to logging
	logEvent(tenantID string, eventType string, message string) error
	clearLog() error
	getEventLog() (logEntries []*types.LogEntry, err error)

	// interfaces related to workloads
	getCNCIWorkloadID() (id string, err error)
	getWorkloadNoCache(id string) (*workload, error)
	getWorkloadsNoCache() ([]*workload, error)

	// interfaces related to tenants
	addLimit(tenantID string, resourceID int, limit int) (err error)
	addTenant(id string, MAC string) (err error)
	getTenantNoCache(id string) (t *tenant, err error)
	getTenantsNoCache() ([]*tenant, error)
	updateTenant(t *tenant) (err error)
	releaseTenantIP(tenantID string, subnetInt int, rest int) (err error)
	claimTenantIP(tenantID string, subnetInt int, rest int) (err error)

	// interfaces related to instances
	getInstances() (instances []*types.Instance, err error)
	addInstance(instance *types.Instance) (err error)
	removeInstance(instanceID string) (err error)
	// this should be merged into removeInstance
	deleteUsageNoCache(instanceID string) (err error)

	// interfaces related to statistics
	addNodeStatDB(stat payloads.Stat) (err error)
	getNodeSummary() (Summary []*types.NodeSummary, err error)
	addInstanceStatsDB(stats []payloads.InstanceStat, nodeID string) (err error)
	addFrameStat(stat payloads.FrameTrace) (err error)
	getBatchFrameSummary() (stats []types.BatchFrameSummary, err error)
	getBatchFrameStatistics(label string) (stats []types.BatchFrameStat, err error)
}

// Datastore provides context for the datastore package.
type Datastore struct {
	db persistentStore

	cnciAddedChans map[string]chan bool
	cnciAddedLock  *sync.Mutex

	nodeLastStat     map[string]payloads.CiaoComputeNode
	nodeLastStatLock *sync.RWMutex

	instanceLastStat     map[string]payloads.CiaoServerStats
	instanceLastStatLock *sync.RWMutex

	tenants     map[string]*tenant
	tenantsLock *sync.RWMutex
	allSubnets  map[int]bool

	workloads      map[string]*workload
	workloadsLock  *sync.RWMutex
	cnciWorkloadID string

	nodes     map[string]*node
	nodesLock *sync.RWMutex

	instances     map[string]*types.Instance
	instancesLock *sync.RWMutex

	tenantUsage     map[string][]payloads.CiaoUsage
	tenantUsageLock *sync.RWMutex
}

// Init initializes the private data for the Datastore object.
// The sql tables are populated with initial data from csv
// files if this is the first time the database has been
// created.  The datastore caches are also filled.
func (ds *Datastore) Init(config Config) (err error) {
	// init persistentStore first...
	ds.db, err = getPersistentStore(config)
	if err != nil {
		return
	}

	ds.cnciAddedChans = make(map[string]chan bool)
	ds.cnciAddedLock = &sync.Mutex{}

	ds.nodeLastStat = make(map[string]payloads.CiaoComputeNode)
	ds.nodeLastStatLock = &sync.RWMutex{}

	ds.instanceLastStat = make(map[string]payloads.CiaoServerStats)
	ds.instanceLastStatLock = &sync.RWMutex{}

	// warning, do not use the tenant cache to get
	// networking information right now.  that is not
	// updated, just the resources
	ds.tenants = make(map[string]*tenant)
	ds.allSubnets = make(map[int]bool)
	ds.tenantsLock = &sync.RWMutex{}

	// cache all our instances prior to getting tenants
	ds.instancesLock = &sync.RWMutex{}
	ds.instances = make(map[string]*types.Instance)

	instances, err := ds.db.getInstances()
	if err != nil {
		glog.Warning(err)
	} else {
		for i := range instances {
			ds.instances[instances[i].Id] = instances[i]
		}
	}

	// cache our current tenants into a map that we can
	// quickly index
	tenants, err := ds.getTenants()
	if err == nil {
		for i := range tenants {
			ds.tenants[tenants[i].Id] = tenants[i]
		}
	}

	// cache the workloads into a map so that we can
	// quickly index
	ds.workloadsLock = &sync.RWMutex{}
	ds.workloads = make(map[string]*workload)
	workloads, err := ds.getWorkloads()
	if err != nil {
		glog.Warning(err)
	} else {
		for i := range workloads {
			ds.workloads[workloads[i].Id] = workloads[i]
		}
	}

	ds.cnciWorkloadID, err = ds.db.getCNCIWorkloadID()
	if err != nil {
		glog.Warning(err)
	}

	ds.nodesLock = &sync.RWMutex{}
	ds.nodes = make(map[string]*node)

	for key, i := range ds.instances {
		n, ok := ds.nodes[i.NodeId]
		if !ok {
			newNode := types.Node{
				ID: i.NodeId,
			}
			n = &node{
				Node:      newNode,
				instances: make(map[string]*types.Instance),
			}
			ds.nodes[i.NodeId] = n
		}
		ds.nodes[i.NodeId].instances[key] = i
	}

	ds.tenantUsage = make(map[string][]payloads.CiaoUsage)
	ds.tenantUsageLock = &sync.RWMutex{}

	return
}

func (ds *Datastore) Exit() {
	ds.db.disconnect()
}

// AddTenantChan allows a caller to pass in a channel for CNCI Launch status.
// When a CNCI has been added to the datastore and a channel exists,
// success will be indicated on the channel.  If a CNCI failure occurred
// and a channel exists, failure will be indicated on the channel.
func (ds *Datastore) AddTenantChan(c chan bool, tenantID string) {
	ds.cnciAddedLock.Lock()
	ds.cnciAddedChans[tenantID] = c
	ds.cnciAddedLock.Unlock()
}

// AddLimit allows the caller to store a limt for a specific resource for a tenant.
func (ds *Datastore) AddLimit(tenantID string, resourceID int, limit int) (err error) {
	err = ds.db.addLimit(tenantID, resourceID, limit)
	if err != nil {
		return
	}

	// update cache
	ds.tenantsLock.Lock()
	tenant := ds.tenants[tenantID]
	if tenant != nil {
		resources := tenant.Resources
		for i := range resources {
			if resources[i].Rtype == resourceID {
				resources[i].Limit = limit
				break
			}
		}
	}
	ds.tenantsLock.Unlock()
	return
}

func newHardwareAddr() (hw net.HardwareAddr, err error) {
	buf := make([]byte, 6)
	_, err = rand.Read(buf)
	if err != nil {
		return
	}

	// vnic creation seems to require not just the
	// bit 1 to be set, but the entire byte to be
	// set to 2.  Also, ensure that we get no
	// overlap with tenant mac addresses by not allowing
	// byte 1 to ever be zero.
	buf[0] = 2
	if buf[1] == 0 {
		buf[1] = 3
	}
	hw = net.HardwareAddr(buf)
	return
}

// AddTenant stores information about a tenant into the datastore.
// it creates a MAC address for the tenant network and makes sure
// that this new tenant is cached.
func (ds *Datastore) AddTenant(id string) (tenant *types.Tenant, err error) {
	hw, err := newHardwareAddr()
	if err != nil {
		glog.V(2).Info("error creating mac address", err)
		return
	}

	err = ds.db.addTenant(id, hw.String())

	t, err := ds.getTenant(id)
	if err != nil || t == nil {
		glog.V(2).Info(err, " unable to get tenant: ", id)
	}

	ds.tenantsLock.Lock()
	ds.tenants[id] = t
	ds.tenantsLock.Unlock()

	return &t.Tenant, err
}

func (ds *Datastore) getTenant(id string) (t *tenant, err error) {
	// check cache first
	ds.tenantsLock.RLock()
	t = ds.tenants[id]
	ds.tenantsLock.RUnlock()

	if t != nil {
		return
	}

	return ds.db.getTenantNoCache(id)
}

// GetTenant returns details about a tenant referenced by the uuid
func (ds *Datastore) GetTenant(id string) (tenant *types.Tenant, err error) {
	t, err := ds.getTenant(id)
	if err != nil || t == nil {
		return nil, err
	}

	return &t.Tenant, nil
}

func (ds *Datastore) getWorkload(id string) (*workload, error) {
	// check the cache first
	ds.workloadsLock.RLock()
	wl := ds.workloads[id]
	ds.workloadsLock.RUnlock()

	if wl != nil {
		return wl, nil
	}

	return ds.db.getWorkloadNoCache(id)
}

// GetWorkload returns details about a specific workload referenced by id
func (ds *Datastore) GetWorkload(id string) (*types.Workload, error) {
	wl, err := ds.getWorkload(id)
	if err != nil {
		return nil, err
	}

	return &wl.Workload, nil
}

func (ds *Datastore) getWorkloads() ([]*workload, error) {
	var workloads []*workload

	// check the cache first
	ds.workloadsLock.RLock()
	if len(ds.workloads) > 0 {
		for _, wl := range ds.workloads {
			workloads = append(workloads, wl)
		}
		ds.workloadsLock.RUnlock()
		return workloads, nil
	}
	ds.workloadsLock.RUnlock()

	return ds.db.getWorkloadsNoCache()
}

// GetWorkloads returns all known tenant workloads
func (ds *Datastore) GetWorkloads() ([]*types.Workload, error) {
	var workloads []*types.Workload

	// yes, we have loop through all the workloads twice
	// now.  We can revisit this later if it proves to
	// be something we should optimize, but for now I
	// think it'd be better to reuse the code.
	wls, err := ds.getWorkloads()
	if err != nil {
		return nil, err
	}

	if len(wls) > 0 {
		for _, wl := range wls {
			workloads = append(workloads, &wl.Workload)
		}
	}

	return workloads, nil
}

// AddCNCIIP will associate a new IP address with an existing CNCI
// via the mac address
func (ds *Datastore) AddCNCIIP(cnciMAC string, ip string) (err error) {
	ds.tenantsLock.Lock()

	var ok bool
	var tenantID string
	var tenant *tenant

	for tenantID, tenant = range ds.tenants {
		if tenant.CNCIMAC == cnciMAC {
			ok = true
			break
		}
	}

	if !ok {
		ds.tenantsLock.Unlock()
		return errors.New("No Tenant")
	}

	tenant.CNCIIP = ip

	ds.tenantsLock.Unlock()

	// not sure what to do about an error
	_ = ds.db.updateTenant(tenant)

	ds.cnciAddedLock.Lock()

	c, ok := ds.cnciAddedChans[tenantID]
	if ok {
		delete(ds.cnciAddedChans, tenantID)
	}

	ds.cnciAddedLock.Unlock()

	if c != nil {
		c <- true
	}

	return
}

// AddTenantCNCI will associate a new CNCI instance with a specific tenant.
// The instanceID of the new CNCI instance and the MAC address of the new instance
// are stored in the sql database and updated in the cache.
func (ds *Datastore) AddTenantCNCI(tenantID string, instanceID string, mac string) (err error) {
	// update tenants cache
	ds.tenantsLock.Lock()

	tenant, ok := ds.tenants[tenantID]
	if !ok {
		ds.tenantsLock.Unlock()
		return errors.New("No Tenant")
	}

	tenant.CNCIID = instanceID
	tenant.CNCIMAC = mac

	ds.tenantsLock.Unlock()

	return ds.db.updateTenant(tenant)
}

func (ds *Datastore) removeTenantCNCI(tenantID string) (err error) {
	// update tenants cache
	ds.tenantsLock.Lock()

	tenant, ok := ds.tenants[tenantID]
	if !ok {
		ds.tenantsLock.Unlock()
		return errors.New("No Tenant")
	}

	tenant.CNCIID = ""
	tenant.CNCIIP = ""

	ds.tenantsLock.Unlock()

	return ds.db.updateTenant(tenant)
}

func (ds *Datastore) getTenants() ([]*tenant, error) {
	var tenants []*tenant

	// check the cache first
	ds.tenantsLock.RLock()
	if len(ds.tenants) > 0 {
		for _, value := range ds.tenants {
			tenants = append(tenants, value)
		}
		ds.tenantsLock.RUnlock()
		return tenants, nil
	}
	ds.tenantsLock.RUnlock()

	return ds.db.getTenantsNoCache()
}

// GetAllTenants returns all the tenants from the datastore.
func (ds *Datastore) GetAllTenants() ([]*types.Tenant, error) {
	var tenants []*types.Tenant

	// yes, this makes it so we have to loop through
	// tenants twice, but there probably aren't huge
	// numbers of tenants. I'd rather reuse the code
	// than make this more efficient at this point.
	ts, err := ds.getTenants()

	if err != nil {
		return nil, err
	}

	if len(ts) > 0 {
		for _, value := range ts {
			tenants = append(tenants, &value.Tenant)
		}
	}

	return tenants, nil
}

// ReleaseTenantIP will return an IP address previously allocated to the pool.
// Once a tenant IP address is released, it can be reassigned to another
// instance.
func (ds *Datastore) ReleaseTenantIP(tenantID string, ip string) (err error) {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return errors.New("Invalid IPv4 Address")
	}

	ipBytes := ipAddr.To4()
	if ipBytes == nil {
		return errors.New("Unable to convert ip to bytes")
	}
	subnetInt := binary.BigEndian.Uint16(ipBytes[1:3])

	// clear from cache
	ds.tenantsLock.Lock()
	if ds.tenants[tenantID] != nil {
		ds.tenants[tenantID].network[int(subnetInt)][int(ipBytes[3])] = false
	}
	ds.tenantsLock.Unlock()

	return ds.db.releaseTenantIP(tenantID, int(subnetInt), int(ipBytes[3]))
}

// AllocateTenantIP will find a free IP address within a tenant network.
// For now we make each tenant have unique subnets even though it
// isn't actually needed because of a docker issue.
func (ds *Datastore) AllocateTenantIP(tenantID string) (ip net.IP, err error) {
	var subnetInt uint16
	subnetInt = 0

	ds.tenantsLock.Lock()
	network := ds.tenants[tenantID].network
	subnets := ds.tenants[tenantID].subnets

	// find any subnet assigned to this tenant with available addresses
	sort.Ints(subnets)
	for _, k := range subnets {
		if len(network[k]) < 253 {
			subnetInt = uint16(k)
		}
	}

	var subnetBytes = []byte{16, 0}
	if subnetInt == 0 {
		i := binary.BigEndian.Uint16(subnetBytes)

		for {
			// sub, ok := network[int(i)]
			// for now, prevent overlapping subnets
			// due to bug in docker.
			ok := ds.allSubnets[int(i)]
			if !ok {
				sub := make(map[int]bool)
				network[int(i)] = sub

				// claim so no one else can use it
				ds.allSubnets[int(i)] = true

				break
			}
			if subnetBytes[1] == 255 {
				if subnetBytes[0] == 31 {
					// out of possible subnets
					glog.Warning("Out of Subnets")
					ds.tenantsLock.Unlock()
					return nil, errors.New("Out of subnets")
				}
				subnetBytes[0]++
				subnetBytes[1] = 0
			} else {
				subnetBytes[1]++
			}
			i = binary.BigEndian.Uint16(subnetBytes)
		}
		subnetInt = i
		ds.tenants[tenantID].subnets = append(subnets, int(subnetInt))
	} else {
		binary.BigEndian.PutUint16(subnetBytes, subnetInt)
	}

	hosts := network[int(subnetInt)]

	rest := 2
	for {
		if hosts[rest] == false {
			hosts[rest] = true
			break
		}

		if rest == 255 {
			// this should never happen
			glog.Warning("ran out of host numbers")
			ds.tenantsLock.Unlock()
			return nil, errors.New("rand out of host numbers")
		}
		rest++
	}

	ds.tenantsLock.Unlock()

	go ds.db.claimTenantIP(tenantID, int(subnetInt), rest)

	// convert to IP type.
	next := net.IPv4(172, subnetBytes[0], subnetBytes[1], byte(rest))
	return next, err
}

// GetAllInstances retrieves all instances out of the datastore.
func (ds *Datastore) GetAllInstances() (instances []*types.Instance, err error) {
	// always get from cache
	ds.instancesLock.RLock()
	if len(ds.instances) > 0 {
		for _, val := range ds.instances {
			instances = append(instances, val)
		}
	}
	ds.instancesLock.RUnlock()

	return instances, nil
}

func (ds *Datastore) getInstance(id string) (instance *types.Instance, err error) {
	// always get from cache
	ds.instancesLock.RLock()
	value, ok := ds.instances[id]
	ds.instancesLock.RUnlock()

	if !ok {
		err = errors.New("Instance Not Found")
	}

	return value, err
}

// GetAllInstancesFromTenant will retrieve all instances belonging to a specific tenant
func (ds *Datastore) GetAllInstancesFromTenant(tenantID string) (instances []*types.Instance, err error) {
	ds.tenantsLock.RLock()

	t, ok := ds.tenants[tenantID]
	if ok {
		for _, val := range t.instances {
			instances = append(instances, val)
		}

		ds.tenantsLock.RUnlock()

		return
	}

	ds.tenantsLock.RUnlock()

	return nil, nil
}

// GetAllInstancesByNode will retrieve all the instances running on a specific compute Node.
func (ds *Datastore) GetAllInstancesByNode(nodeID string) (instances []*types.Instance, err error) {
	ds.nodesLock.RLock()

	n, ok := ds.nodes[nodeID]
	if ok {
		for _, val := range n.instances {
			instances = append(instances, val)
		}
	}

	ds.nodesLock.RUnlock()

	return instances, nil
}

// GetInstanceFromTenant will be replaced soon with something else that makes more sense.
// this function doesn't make sense because if we know the instanceID
// we really don't care what the tenantID is.
func (ds *Datastore) GetInstanceFromTenant(tenantID string, instanceID string) (*types.Instance, error) {
	return ds.getInstance(instanceID)
}

// AddInstance will store a new instance in the datastore.
// The instance will be updated both in the cache and in the database
func (ds *Datastore) AddInstance(instance *types.Instance) (err error) {
	// add to cache
	ds.instancesLock.Lock()

	ds.instances[instance.Id] = instance

	instanceStat := payloads.CiaoServerStats{
		ID:        instance.Id,
		TenantID:  instance.TenantId,
		NodeID:    instance.NodeId,
		Timestamp: time.Now(),
		Status:    instance.State,
	}

	ds.instanceLastStatLock.Lock()
	ds.instanceLastStat[instance.Id] = instanceStat
	ds.instanceLastStatLock.Unlock()

	ds.instancesLock.Unlock()

	ds.tenantsLock.Lock()

	tenant := ds.tenants[instance.TenantId]
	if tenant != nil {
		for name, val := range instance.Usage {
			for i := range tenant.Resources {
				if tenant.Resources[i].Rname == name {
					tenant.Resources[i].Usage += val
					break
				}
			}
		}

		// increment instances count
		for i := range tenant.Resources {
			if tenant.Resources[i].Rtype == 1 {
				tenant.Resources[i].Usage++
				break
			}
		}
		tenant.instances[instance.Id] = instance
	}

	ds.tenantsLock.Unlock()

	return ds.db.addInstance(instance)
}

// RestartFailure logs a RestartFailure in the datastore
func (ds *Datastore) RestartFailure(instanceID string, reason payloads.RestartFailureReason) (err error) {
	i, err := ds.getInstance(instanceID)
	if err != nil {
		return
	}

	msg := fmt.Sprintf("Restart Failure %s: %s", instanceID, reason.String())
	ds.db.logEvent(i.TenantId, string(userError), msg)
	return
}

// StopFailure logs a StopFailure in the datastore
func (ds *Datastore) StopFailure(instanceID string, reason payloads.StopFailureReason) (err error) {
	i, err := ds.getInstance(instanceID)
	if err != nil {
		return
	}

	msg := fmt.Sprintf("Stop Failure %s: %s", instanceID, reason.String())
	ds.db.logEvent(i.TenantId, string(userError), msg)
	return
}

// StartFailure will clean up after a failure to start an instance.
// If an instance was a CNCI, this function will remove the CNCI instance
// for this tenant. If the instance was a normal tenant instance, the
// IP address will be released and the instance will be deleted from the
// datastore.
func (ds *Datastore) StartFailure(instanceID string, reason payloads.StartFailureReason) (err error) {
	var tenantID string
	var cnci bool

	ds.tenantsLock.RLock()
	for key, t := range ds.tenants {
		if t.CNCIID == instanceID {
			cnci = true
			tenantID = key
			break
		}
	}
	ds.tenantsLock.RUnlock()

	if cnci == true {
		glog.Warning("CNCI ", instanceID, " Failed to start")

		err = ds.removeTenantCNCI(tenantID)
		if err != nil {
			glog.Warning(err)
		}
		msg := fmt.Sprintf("CNCI Start Failure %s: %s", instanceID, reason.String())
		ds.db.logEvent(tenantID, string(userError), msg)

		ds.cnciAddedLock.Lock()

		c, ok := ds.cnciAddedChans[tenantID]
		if ok {
			delete(ds.cnciAddedChans, tenantID)
		}

		ds.cnciAddedLock.Unlock()

		if c != nil {
			c <- false
		}

		return
	}

	i, err := ds.getInstance(instanceID)
	if err != nil {
		return
	}

	tenantID = i.TenantId
	ipAddress := i.IPAddress

	switch reason {
	case payloads.FullCloud,
		payloads.FullComputeNode,
		payloads.NoComputeNodes,
		payloads.NoNetworkNodes,
		payloads.InvalidPayload,
		payloads.InvalidData,
		payloads.ImageFailure,
		payloads.NetworkFailure:

		ds.instancesLock.Lock()
		i := ds.instances[instanceID]
		delete(ds.instances, instanceID)
		ds.instancesLock.Unlock()

		ds.tenantsLock.Lock()
		delete(ds.tenants[tenantID].instances, instanceID)
		ds.tenantsLock.Unlock()

		err = ds.deleteAllUsage(i, tenantID)
		if err != nil {
			glog.Warning(err)
		}

		err = ds.db.removeInstance(instanceID)
		if err != nil {
			return err
		}

		err = ds.ReleaseTenantIP(tenantID, ipAddress)
		if err != nil {
			glog.V(2).Info("StartFailure: ", err)
		}
	case payloads.LaunchFailure,
		payloads.AlreadyRunning,
		payloads.InstanceExists:
	}
	msg := fmt.Sprintf("Start Failure %s: %s", instanceID, reason.String())
	ds.db.logEvent(tenantID, string(userError), msg)
	return
}

// DeleteInstance removes an instance from the datastore.
func (ds *Datastore) DeleteInstance(instanceID string) (err error) {
	ds.instanceLastStatLock.Lock()
	delete(ds.instanceLastStat, instanceID)
	ds.instanceLastStatLock.Unlock()

	ds.instancesLock.Lock()
	i := ds.instances[instanceID]
	delete(ds.instances, instanceID)
	ds.instancesLock.Unlock()

	ds.tenantsLock.Lock()
	delete(ds.tenants[i.TenantId].instances, instanceID)
	ds.tenantsLock.Unlock()

	// we may not have received any node stats for this instance
	if i.NodeId != "" {
		ds.nodesLock.Lock()
		delete(ds.nodes[i.NodeId].instances, instanceID)
		ds.nodesLock.Unlock()
	}

	err = ds.deleteAllUsage(i, i.TenantId)
	if err != nil {
		glog.Warning(err)
	}

	err = ds.db.removeInstance(i.Id)
	if err != nil {
		glog.V(2).Info("DeleteInstance: ", err)
	}

	err = ds.ReleaseTenantIP(i.TenantId, i.IPAddress)
	if err != nil {
		glog.V(2).Info("DeleteInstance: ", err)
		return
	}

	msg := fmt.Sprintf("Deleted Instance %s", instanceID)
	ds.db.logEvent(i.TenantId, string(userInfo), msg)
	return
}

// GetInstanceInfo will be replaced by something else soon that makes more sense.
func (ds *Datastore) GetInstanceInfo(instanceID string) (nodeID string, state string, err error) {
	instance, err := ds.getInstance(instanceID)
	if err != nil {
		return
	}

	if instance != nil {
		nodeID = instance.NodeId
		state = instance.State
	}

	return
}

func (ds *Datastore) deleteAllUsage(i *types.Instance, tenantID string) (err error) {
	// update tenant usage in cache
	ds.tenantsLock.Lock()
	tenant := ds.tenants[tenantID]
	if tenant != nil {
		for name, val := range i.Usage {
			for i := range tenant.Resources {
				if tenant.Resources[i].Rname == name {
					tenant.Resources[i].Usage -= val
					break
				}
			}
		}
		// decrement instances count
		for i := range tenant.Resources {
			if tenant.Resources[i].Rtype == 1 {
				tenant.Resources[i].Usage--
				break
			}
		}
	}
	ds.tenantsLock.Unlock()

	// update persistent store
	return ds.db.deleteUsageNoCache(i.Id)
}

// HandleStats makes sure that the data from the stat payload is stored.
func (ds *Datastore) HandleStats(stat payloads.Stat) (err error) {
	if stat.Load != -1 {
		ds.addNodeStat(stat)
	}

	err = ds.addInstanceStats(stat.Instances, stat.NodeUUID)
	if err != nil {
		glog.Warning(err)
	}

	return
}

// HandleTraceReport stores the provided trace data in the datastore.
func (ds *Datastore) HandleTraceReport(trace payloads.Trace) (err error) {
	for index := range trace.Frames {
		i := trace.Frames[index]
		err = ds.db.addFrameStat(i)
		if err != nil {
			glog.Warning(err)
		}
	}
	return nil
}

// GetInstanceLastStats retrieves the last instances stats recieved for this node.
// It returns it in a format suitable for the compute API.
func (ds *Datastore) GetInstanceLastStats(nodeID string) payloads.CiaoServersStats {
	var serversStats payloads.CiaoServersStats

	ds.instanceLastStatLock.RLock()
	for _, instance := range ds.instanceLastStat {
		if instance.NodeID != nodeID {
			continue
		}
		serversStats.Servers = append(serversStats.Servers, instance)
	}
	ds.instanceLastStatLock.RUnlock()

	return serversStats
}

// GetNodeLastStats retrieves the last nodes stats recieved for this node.
// It returns it in a format suitable for the compute API.
func (ds *Datastore) GetNodeLastStats() payloads.CiaoComputeNodes {
	var computeNodes payloads.CiaoComputeNodes

	ds.nodeLastStatLock.RLock()
	for _, node := range ds.nodeLastStat {
		computeNodes.Nodes = append(computeNodes.Nodes, node)
	}
	ds.nodeLastStatLock.RUnlock()

	return computeNodes
}

func (ds *Datastore) addNodeStat(stat payloads.Stat) (err error) {
	ds.nodesLock.Lock()
	n, ok := ds.nodes[stat.NodeUUID]
	if !ok {
		n = &node{}
		n.instances = make(map[string]*types.Instance)
		ds.nodes[stat.NodeUUID] = n
	}
	n.ID = stat.NodeUUID
	n.Hostname = stat.NodeHostName
	ds.nodesLock.Unlock()

	cnStat := payloads.CiaoComputeNode{
		ID:            stat.NodeUUID,
		Status:        stat.Status,
		Load:          stat.Load,
		MemTotal:      stat.MemTotalMB,
		MemAvailable:  stat.MemAvailableMB,
		DiskTotal:     stat.DiskTotalMB,
		DiskAvailable: stat.DiskAvailableMB,
		OnlineCPUs:    stat.CpusOnline,
	}

	ds.nodeLastStatLock.Lock()

	delete(ds.nodeLastStat, stat.NodeUUID)
	ds.nodeLastStat[stat.NodeUUID] = cnStat

	ds.nodeLastStatLock.Unlock()

	return ds.db.addNodeStatDB(stat)
}

var tenantUsagePeriodMinutes float64 = 5

func (ds *Datastore) updateTenantUsageNeeded(delta payloads.CiaoUsage, tenantID string) bool {
	if delta.VCPU == 0 &&
		delta.Memory == 0 &&
		delta.Disk == 0 {
		return false
	}

	return true
}

func (ds *Datastore) updateTenantUsage(delta payloads.CiaoUsage, tenantID string) {
	if ds.updateTenantUsageNeeded(delta, tenantID) == false {
		return
	}

	createNewUsage := true
	lastUsage := payloads.CiaoUsage{}

	ds.tenantUsageLock.Lock()

	tenantUsage := ds.tenantUsage[tenantID]
	if len(tenantUsage) != 0 {
		lastUsage = tenantUsage[len(tenantUsage)-1]
		// We will not create more than one entry per tenant every tenantUsagePeriodMinutes
		if time.Since(lastUsage.Timestamp).Minutes() < tenantUsagePeriodMinutes {
			createNewUsage = false
		}
	}

	newUsage := payloads.CiaoUsage{
		VCPU:   lastUsage.VCPU + delta.VCPU,
		Memory: lastUsage.Memory + delta.Memory,
		Disk:   lastUsage.Disk + delta.Disk,
	}

	// If we need to create a new usage entry, we timestamp it now.
	// If not we just update the last entry.
	if createNewUsage == true {
		newUsage.Timestamp = time.Now()
		ds.tenantUsage[tenantID] = append(ds.tenantUsage[tenantID], newUsage)
	} else {
		newUsage.Timestamp = lastUsage.Timestamp
		tenantUsage[len(tenantUsage)-1] = newUsage
	}

	ds.tenantUsageLock.Unlock()
}

func (ds *Datastore) GetTenantUsage(tenantID string, start time.Time, end time.Time) ([]payloads.CiaoUsage, error) {
	ds.tenantUsageLock.RLock()
	defer ds.tenantUsageLock.RUnlock()

	tenantUsage := ds.tenantUsage[tenantID]
	if tenantUsage == nil || len(tenantUsage) == 0 {
		return nil, fmt.Errorf("No usage history for %s", tenantID)
	}

	historyLength := len(tenantUsage)
	if tenantUsage[0].Timestamp.After(end) == true ||
		start.After(tenantUsage[historyLength-1].Timestamp) == true {
		return nil, nil
	}

	first := 0
	last := 0
	for _, u := range tenantUsage {
		if start.After(u.Timestamp) == true {
			first++
		}

		if end.After(u.Timestamp) == true {
			last++
		}
	}

	return tenantUsage[first:last], nil
}

func reduceToZero(v int) int {
	if v < 0 {
		return 0
	}

	return v
}

func (ds *Datastore) addInstanceStats(stats []payloads.InstanceStat, nodeID string) (err error) {
	for index := range stats {
		stat := stats[index]

		instanceStat := payloads.CiaoServerStats{
			ID:        stat.InstanceUUID,
			NodeID:    nodeID,
			Timestamp: time.Now(),
			Status:    stat.State,
			VCPUUsage: reduceToZero(stat.CPUUsage),
			MemUsage:  reduceToZero(stat.MemoryUsageMB),
			DiskUsage: reduceToZero(stat.DiskUsageMB),
		}

		ds.instanceLastStatLock.Lock()

		lastInstanceStat := ds.instanceLastStat[stat.InstanceUUID]

		deltaUsage := payloads.CiaoUsage{
			VCPU:   instanceStat.VCPUUsage - lastInstanceStat.VCPUUsage,
			Memory: instanceStat.MemUsage - lastInstanceStat.MemUsage,
			Disk:   instanceStat.DiskUsage - lastInstanceStat.DiskUsage,
		}

		go ds.updateTenantUsage(deltaUsage, lastInstanceStat.TenantID)

		instanceStat.TenantID = lastInstanceStat.TenantID

		delete(ds.instanceLastStat, stat.InstanceUUID)
		ds.instanceLastStat[stat.InstanceUUID] = instanceStat

		ds.instanceLastStatLock.Unlock()

		ds.instancesLock.Lock()
		instance, ok := ds.instances[stat.InstanceUUID]
		if ok {
			instance.State = stat.State
			instance.NodeId = nodeID
			instance.SSHIP = stat.SSHIP
			instance.SSHPort = stat.SSHPort
			ds.nodesLock.Lock()
			ds.nodes[nodeID].instances[instance.Id] = instance
			ds.nodesLock.Unlock()
		}
		ds.instancesLock.Unlock()
	}

	return ds.db.addInstanceStatsDB(stats, nodeID)
}

// GetTenantCNCISummary retrieves information about a given CNCI id, or all CNCIs
// If the cnci string is the null string, then this function will retrieve all
// tenants.  If cnci is not null, it will only provide information about a specific
// cnci.
func (ds *Datastore) GetTenantCNCISummary(cnci string) (cncis []types.TenantCNCI, err error) {
	cncis = make([]types.TenantCNCI, 0)
	subnetBytes := []byte{0, 0}

	ds.tenantsLock.RLock()

	for _, t := range ds.tenants {
		if cnci != "" && cnci != t.CNCIID {
			continue
		}

		cn := types.TenantCNCI{
			TenantID:   t.Id,
			IPAddress:  t.CNCIIP,
			MACAddress: t.CNCIMAC,
			InstanceID: t.CNCIID,
		}

		for _, subnet := range t.subnets {
			binary.BigEndian.PutUint16(subnetBytes, (uint16)(subnet))
			cn.Subnets = append(cn.Subnets, fmt.Sprintf("Subnet 172.%d.%d.0/8", subnetBytes[0], subnetBytes[1]))
		}

		cncis = append(cncis, cn)

		if cnci != "" && cnci == t.CNCIID {
			break
		}
	}

	ds.tenantsLock.RUnlock()

	return cncis, err
}

// GetCNCIWorkloadID returns the UUID of the workload template
// for the CNCI workload
func (ds *Datastore) GetCNCIWorkloadID() (id string, err error) {
	if ds.cnciWorkloadID == "" {
		return "", errors.New("No CNCI Workload in datastore")
	}

	return ds.cnciWorkloadID, nil
}

// GetNodeSummary provides a summary the state and count of instances running per node.
func (ds *Datastore) GetNodeSummary() (Summary []*types.NodeSummary, err error) {
	// TBD: write a new routine that grabs the node summary info
	// from the cache rather than do this lengthy sql query.
	return ds.db.getNodeSummary()
}

// GetBatchFrameSummary will retieve the count of traces we have for a specific label
func (ds *Datastore) GetBatchFrameSummary() (stats []types.BatchFrameSummary, err error) {
	// until we start caching frame stats, we have to send this
	// right through to the database.
	return ds.db.getBatchFrameSummary()
}

// GetBatchFrameStatistics will show individual trace data per instance for a batch of trace data.
// The batch is identified by the label.
func (ds *Datastore) GetBatchFrameStatistics(label string) (stats []types.BatchFrameStat, err error) {
	// until we start caching frame stats, we have to send this
	// right through to the database.
	return ds.db.getBatchFrameStatistics(label)
}

// GetEventLog retrieves all the log entries stored in the datastore.
func (ds *Datastore) GetEventLog() (logEntries []*types.LogEntry, err error) {
	// we don't as of yet cache any of the events that are logged.
	return ds.db.getEventLog()
}

// ClearLog will remove all the event entries from the event log
func (ds *Datastore) ClearLog() error {
	// we don't as of yet cache any of the events that are logged.
	return ds.db.clearLog()
}
