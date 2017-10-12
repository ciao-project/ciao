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

package datastore

import (
	"fmt"

	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/payloads"
)

// MemoryDB is a memory backed persistentStore implementation for unit testing
type MemoryDB struct {
	tenants         map[string]*tenant
	nodes           map[string]*node
	instances       map[string]*types.Instance
	tenantUsage     map[string][]types.CiaoUsage
	blockDevices    map[string]types.BlockData
	attachments     map[string]types.StorageAttachment
	instanceVolumes map[attachment]string
	logEntries      []*types.LogEntry

	workloadsPath string
}

func (db *MemoryDB) fillWorkloads() error {
	config := types.TenantConfig{
		Name:       "",
		SubnetBits: 24,
	}

	// add dummy public tenant.
	return db.addTenant("public", config)
}

func (db *MemoryDB) init(config Config) error {
	db.tenants = make(map[string]*tenant)
	db.nodes = make(map[string]*node)
	db.instances = make(map[string]*types.Instance)
	db.tenantUsage = make(map[string][]types.CiaoUsage)
	db.blockDevices = make(map[string]types.BlockData)
	db.attachments = make(map[string]types.StorageAttachment)
	db.instanceVolumes = make(map[attachment]string)

	db.workloadsPath = config.InitWorkloadsPath
	return db.fillWorkloads()
}

func (db *MemoryDB) disconnect() {

}

func (db *MemoryDB) logEvent(entry types.LogEntry) error {
	db.logEntries = append(db.logEntries, &entry)

	return nil
}

func (db *MemoryDB) clearLog() error {
	db.logEntries = nil
	return nil
}

func (db *MemoryDB) getEventLog() ([]*types.LogEntry, error) {
	return db.logEntries, nil
}

func (db *MemoryDB) addTenant(id string, config types.TenantConfig) error {
	t := &tenant{
		Tenant: types.Tenant{
			ID:         id,
			Name:       config.Name,
			SubnetBits: config.SubnetBits,
		},
		network:   make(map[int]map[int]bool),
		instances: make(map[string]*types.Instance),
		devices:   make(map[string]types.BlockData),
	}
	db.tenants[id] = t
	return nil
}

func (db *MemoryDB) getTenant(id string) (*tenant, error) {
	tenant, ok := db.tenants[id]
	if !ok {
		return nil, fmt.Errorf("Tenant %s not found", id)
	}
	return tenant, nil
}

func (db *MemoryDB) getTenants() ([]*tenant, error) {
	var tenants []*tenant
	for _, t := range db.tenants {
		tenants = append(tenants, t)
	}
	return tenants, nil
}

func (db *MemoryDB) releaseTenantIP(tenantID string, subnetInt int, rest int) error {
	return nil
}

func (db *MemoryDB) claimTenantIP(tenantID string, subnetInt int, rest int) error {
	return nil
}

func (db *MemoryDB) getInstances() ([]*types.Instance, error) {
	var instances []*types.Instance
	for _, instance := range db.instances {
		instances = append(instances, instance)
	}
	return instances, nil
}

func (db *MemoryDB) addInstance(instance *types.Instance) error {
	return nil
}

func (db *MemoryDB) deleteInstance(instanceID string) error {
	return nil
}

func (db *MemoryDB) addNodeStat(stat payloads.Stat) error {
	return nil
}

func (db *MemoryDB) addInstanceStats(stats []payloads.InstanceStat, nodeID string) error {
	return nil
}

func (db *MemoryDB) addFrameStat(stat payloads.FrameTrace) error {
	return nil
}

func (db *MemoryDB) getBatchFrameSummary() ([]types.BatchFrameSummary, error) {
	return nil, nil
}

func (db *MemoryDB) getBatchFrameStatistics(label string) ([]types.BatchFrameStat, error) {
	return nil, nil
}

func (db *MemoryDB) getWorkloadStorage(ID string) ([]types.StorageResource, error) {
	return []types.StorageResource{}, nil
}

func (db *MemoryDB) getAllBlockData() (map[string]types.BlockData, error) {
	return db.blockDevices, nil
}

func (db *MemoryDB) addBlockData(data types.BlockData) error {
	return nil
}

func (db *MemoryDB) updateBlockData(types.BlockData) error {
	return nil
}

func (db *MemoryDB) deleteBlockData(string) error {
	return nil
}

func (db *MemoryDB) getTenantDevices(tenantID string) (map[string]types.BlockData, error) {
	return nil, nil
}

func (db *MemoryDB) addStorageAttachment(a types.StorageAttachment) error {
	return nil
}

func (db *MemoryDB) getAllStorageAttachments() (map[string]types.StorageAttachment, error) {
	return db.attachments, nil
}

func (db *MemoryDB) deleteStorageAttachment(ID string) error {
	return nil
}

func (db *MemoryDB) addPool(pool types.Pool) error {
	return nil
}

func (db *MemoryDB) updatePool(pool types.Pool) error {
	return nil
}

func (db *MemoryDB) deletePool(ID string) error {
	return nil
}

func (db *MemoryDB) getAllPools() map[string]types.Pool {
	return make(map[string]types.Pool)
}

func (db *MemoryDB) addMappedIP(m types.MappedIP) error {
	return nil
}

func (db *MemoryDB) deleteMappedIP(ID string) error {
	return nil
}

func (db *MemoryDB) getMappedIPs() map[string]types.MappedIP {
	return make(map[string]types.MappedIP)
}

func (db *MemoryDB) updateWorkload(wl types.Workload) error {
	return nil
}

func (db *MemoryDB) deleteWorkload(ID string) error {
	return nil
}

func (db *MemoryDB) updateQuotas(tenantID string, qds []types.QuotaDetails) error {
	return nil
}

func (db *MemoryDB) getQuotas(tenantID string) ([]types.QuotaDetails, error) {
	return []types.QuotaDetails{}, nil
}

func (db *MemoryDB) updateInstance(instance *types.Instance) error {
	return nil
}

func (db *MemoryDB) updateTenant(tenant *types.Tenant) error {
	return nil
}

func (db *MemoryDB) deleteTenant(tenantID string) error {
	delete(db.tenants, tenantID)
	return nil
}

func (db *MemoryDB) getImages() ([]types.Image, error) {
	return []types.Image{}, nil
}

func (db *MemoryDB) updateImage(i types.Image) error {
	return nil
}

func (db *MemoryDB) deleteImage(ID string) error {
	return nil
}
