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

package datastore

import (
	"reflect"
	"testing"
	"time"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/ciao-storage"
	"github.com/01org/ciao/ssntp/uuid"
)

func TestGetWorkloadStorage(t *testing.T) {
	config := Config{
		PersistentURI: "file:memdb3?mode=memory&cache=shared",
		TransientURI:  "file:memdb4?mode=memory&cache=shared",
	}

	db, err := getPersistentStore(config)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.getWorkloadStorage("validid")
	if err != nil {
		t.Fatal(err)
	}

	db.disconnect()
}

func TestGetTenantDevices(t *testing.T) {
	config := Config{
		PersistentURI: "file:memdb5?mode=memory&cache=shared",
		TransientURI:  "file:memdb6?mode=memory&cache=shared",
	}

	db, err := getPersistentStore(config)
	if err != nil {
		t.Fatal(err)
	}

	blockDevice := storage.BlockDevice{
		ID: uuid.Generate().String(),
	}

	data := types.BlockData{
		BlockDevice: blockDevice,
		Size:        0,
		State:       types.Available,
		TenantID:    uuid.Generate().String(),
		CreateTime:  time.Now(),
	}

	err = db.createBlockData(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure our query works.
	devices, err := db.getTenantDevices(data.TenantID)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := devices[data.ID]
	if !ok {
		t.Fatal("device not in map")
	}

	db.disconnect()
}

func TestGetTenantWithStorage(t *testing.T) {
	config := Config{
		PersistentURI: "file:memdb11?mode=memory&cache=shared",
		TransientURI:  "file:memdb12?mode=memory&cache=shared",
	}

	db, err := getPersistentStore(config)
	if err != nil {
		t.Fatal(err)
	}

	// add a tenant.
	tenantID := uuid.Generate().String()
	mac := "validmac"

	err = db.addTenant(tenantID, mac)
	if err != nil {
		t.Fatal(err)
	}

	blockDevice := storage.BlockDevice{
		ID: uuid.Generate().String(),
	}

	data := types.BlockData{
		BlockDevice: blockDevice,
		Size:        0,
		State:       types.Available,
		TenantID:    tenantID,
		CreateTime:  time.Now(),
	}

	err = db.createBlockData(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure our query works.
	tenant, err := db.getTenantNoCache(data.TenantID)
	if err != nil {
		t.Fatal(err)
	}

	if tenant.devices == nil {
		t.Fatal("devices is nil")
	}

	d := tenant.devices[data.ID]
	if d.ID != data.ID {
		t.Fatal("device not correct")
	}

	db.disconnect()
}

func TestGetAllBlockData(t *testing.T) {
	config := Config{
		PersistentURI: "file:memdb7?mode=memory&cache=shared",
		TransientURI:  "file:memdb8?mode=memory&cache=shared",
	}

	db, err := getPersistentStore(config)
	if err != nil {
		t.Fatal(err)
	}

	blockDevice := storage.BlockDevice{
		ID: uuid.Generate().String(),
	}

	data := types.BlockData{
		BlockDevice: blockDevice,
		Size:        0,
		State:       types.Available,
		TenantID:    uuid.Generate().String(),
		CreateTime:  time.Now(),
	}

	err = db.createBlockData(data)
	if err != nil {
		t.Fatal(err)
	}

	devices, err := db.getAllBlockData()
	if err != nil {
		t.Fatal(err)
	}

	_, ok := devices[data.ID]
	if !ok {
		t.Fatal(err)
	}

	db.disconnect()
}

func TestDeleteBlockData(t *testing.T) {
	config := Config{
		PersistentURI: "file:DeleteBlockData1?mode=memory&cache=shared",
		TransientURI:  "file:DeleteBlockData2?mode=memory&cache=shared",
	}

	db, err := getPersistentStore(config)
	if err != nil {
		t.Fatal(err)
	}

	blockDevice := storage.BlockDevice{
		ID: uuid.Generate().String(),
	}

	data := types.BlockData{
		BlockDevice: blockDevice,
		Size:        0,
		State:       types.Available,
		TenantID:    uuid.Generate().String(),
		CreateTime:  time.Now(),
	}

	err = db.createBlockData(data)
	if err != nil {
		t.Fatal(err)
	}

	err = db.deleteBlockData(data.ID)
	if err != nil {
		t.Fatal(err)
	}

	devices, err := db.getAllBlockData()
	if err != nil {
		t.Fatal(err)
	}

	_, ok := devices[data.ID]
	if ok {
		t.Fatal("block devices not deleted")
	}

	db.disconnect()
}

func TestGetAllStorageAttachments(t *testing.T) {
	config := Config{
		PersistentURI: "file:memdb9?mode=memory&cache=shared",
		TransientURI:  "file:memdb10?mode=memory&cache=shared",
	}

	db, err := getPersistentStore(config)
	if err != nil {
		t.Fatal(err)
	}

	a := types.StorageAttachment{
		ID:         uuid.Generate().String(),
		InstanceID: uuid.Generate().String(),
		BlockID:    uuid.Generate().String(),
		Ephemeral:  false,
	}

	err = db.createStorageAttachment(a)
	if err != nil {
		t.Fatal(err)
	}

	attachments, err := db.getAllStorageAttachments()
	if err != nil {
		t.Fatal(err)
	}

	if len(attachments) != 1 {
		t.Fatal(err)
	}

	alpha := attachments[a.ID]

	if alpha != a {
		t.Fatal("Attachment from DB doesn't match original attachment")
	}

	b := types.StorageAttachment{
		ID:         uuid.Generate().String(),
		InstanceID: uuid.Generate().String(),
		BlockID:    uuid.Generate().String(),
		Ephemeral:  true,
	}

	err = db.createStorageAttachment(b)
	if err != nil {
		t.Fatal(err)
	}

	attachments, err = db.getAllStorageAttachments()
	if err != nil {
		t.Fatal(err)
	}

	if len(attachments) != 2 {
		t.Fatal(err)
	}

	err = db.deleteStorageAttachment(a.ID)
	if err != nil {
		t.Fatal(err)
	}

	attachments, err = db.getAllStorageAttachments()
	if err != nil {
		t.Fatal(err)
	}

	if len(attachments) != 1 {
		t.Fatal(err)
	}

	beta := attachments[b.ID]

	if beta != b {
		t.Fatal("Attachment from DB doesn't match original attachment")
	}
	db.disconnect()
}

func TestCreatePool(t *testing.T) {
	config := Config{
		PersistentURI: "file:testcreatepool?mode=memory&cache=shared",
		TransientURI:  "file:testcreatepoolt?mode=memory&cache=shared",
	}

	db, err := getPersistentStore(config)
	if err != nil {
		t.Fatal(err)
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: "test",
	}

	err = db.createPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pools := db.getAllPools()
	if pools == nil {
		t.Fatal("pool not stored")
	}

	p, ok := pools[pool.ID]
	if !ok || (p.Name != "test") {
		t.Fatal("pool not stored")
	}

	db.disconnect()
}

func TestUpdatePool(t *testing.T) {
	config := Config{
		PersistentURI: "file:testupdatepool?mode=memory&cache=shared",
		TransientURI:  "file:testupdatepoolt?mode=memory&cache=shared",
	}

	db, err := getPersistentStore(config)
	if err != nil {
		t.Fatal(err)
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: "test",
	}

	err = db.createPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pool.Free = 2
	pool.TotalIPs = 10

	err = db.updatePool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pools := db.getAllPools()
	if pools == nil {
		t.Fatal("pool not stored")
	}

	p, ok := pools[pool.ID]
	if !ok || p.Free != 2 || p.TotalIPs != 10 {
		t.Fatal("pool not updated")
	}

	db.disconnect()
}

func TestDeletePool(t *testing.T) {
	config := Config{
		PersistentURI: "file:testdeletepool?mode=memory&cache=shared",
		TransientURI:  "file:testdeletepoolt?mode=memory&cache=shared",
	}

	db, err := getPersistentStore(config)
	if err != nil {
		t.Fatal(err)
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: "test",
	}

	err = db.createPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pools := db.getAllPools()
	if pools == nil {
		t.Fatal("pool not stored")
	}

	_, ok := pools[pool.ID]
	if !ok {
		t.Fatal("pool not updated")
	}

	err = db.deletePool(pool.ID)
	if err != nil {
		t.Fatal("pool not deleted")
	}

	pools = db.getAllPools()
	if pools == nil {
		t.Fatal("pool not stored")
	}

	_, ok = pools[pool.ID]
	if ok {
		t.Fatal("pool not deleted")
	}

	db.disconnect()
}

func TestCreateSubnet(t *testing.T) {
	config := Config{
		PersistentURI: "file:testcreatesubnet?mode=memory&cache=shared",
		TransientURI:  "file:testcreatesubnett?mode=memory&cache=shared",
	}

	db, err := getPersistentStore(config)
	if err != nil {
		t.Fatal(err)
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: "test",
	}

	err = db.createPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	subnet := types.ExternalSubnet{
		ID:   uuid.Generate().String(),
		CIDR: "192.168.0.0/24",
	}

	pool.Subnets = append(pool.Subnets, subnet)

	err = db.updatePool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pools := db.getAllPools()
	if pools == nil {
		t.Fatal("pool not stored")
	}

	p, ok := pools[pool.ID]
	if !ok || (p.Name != "test") {
		t.Fatal("pool not stored")
	}

	subs := p.Subnets
	if len(subs) != 1 {
		t.Fatal("subnet not saved")
	}

	if subs[0].CIDR != subnet.CIDR || subs[0].ID != subnet.ID {
		t.Fatal("subnet not saved correctly")
	}

	db.disconnect()
}

func TestDeleteSubnet(t *testing.T) {
	config := Config{
		PersistentURI: "file:testdeletesubnet?mode=memory&cache=shared",
		TransientURI:  "file:testdeletesubnett?mode=memory&cache=shared",
	}

	db, err := getPersistentStore(config)
	if err != nil {
		t.Fatal(err)
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: "test",
	}

	subnet := types.ExternalSubnet{
		ID:   uuid.Generate().String(),
		CIDR: "192.168.0.0/24",
	}

	pool.Subnets = append(pool.Subnets, subnet)

	err = db.createPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pool.Subnets = []types.ExternalSubnet{}
	err = db.updatePool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pools := db.getAllPools()
	if pools == nil {
		t.Fatal("pool not stored")
	}

	p, ok := pools[pool.ID]
	if !ok || (p.Name != "test") {
		t.Fatal("pool not stored")
	}

	subs := p.Subnets
	if len(subs) != 0 {
		t.Fatal("subnet not deleted")
	}

	db.disconnect()
}

func TestCreateAddress(t *testing.T) {
	config := Config{
		PersistentURI: "file:createaddress?mode=memory&cache=shared",
		TransientURI:  "file:createaddresst?mode=memory&cache=shared",
	}

	db, err := getPersistentStore(config)
	if err != nil {
		t.Fatal(err)
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: "test",
	}

	IP := types.ExternalIP{
		ID:      uuid.Generate().String(),
		Address: "192.168.0.1",
	}

	pool.IPs = append(pool.IPs, IP)

	err = db.createPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pools := db.getAllPools()
	if pools == nil {
		t.Fatal("pool not stored")
	}

	p, ok := pools[pool.ID]
	if !ok || (p.Name != "test") {
		t.Fatal("pool not stored")
	}

	addrs := p.IPs
	if len(addrs) != 1 || addrs[0].ID != IP.ID || addrs[0].Address != IP.Address {
		t.Fatal("address not stored correctly")
	}

	db.disconnect()
}

func TestDeleteAddress(t *testing.T) {
	config := Config{
		PersistentURI: "file:deleteaddress?mode=memory&cache=shared",
		TransientURI:  "file:deleteaddresst?mode=memory&cache=shared",
	}

	db, err := getPersistentStore(config)
	if err != nil {
		t.Fatal(err)
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: "test",
	}

	IP := types.ExternalIP{
		ID:      uuid.Generate().String(),
		Address: "192.168.0.1",
	}

	pool.IPs = append(pool.IPs, IP)

	err = db.createPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pool.IPs = []types.ExternalIP{}

	err = db.updatePool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pools := db.getAllPools()
	if pools == nil {
		t.Fatal("pool not stored")
	}

	p, ok := pools[pool.ID]
	if !ok || (p.Name != "test") {
		t.Fatal("pool not stored")
	}

	addrs := p.IPs
	if len(addrs) != 0 {
		t.Fatal("address not deleted")
	}

	db.disconnect()
}

func TestCreateMappedIP(t *testing.T) {
	config := Config{
		PersistentURI: "file:createmappedaddress?mode=memory&cache=shared",
		TransientURI:  "file:createmappedaddresst?mode=memory&cache=shared",
	}

	db, err := getPersistentStore(config)
	if err != nil {
		t.Fatal(err)
	}

	i := types.Instance{
		ID:         uuid.Generate().String(),
		TenantID:   uuid.Generate().String(),
		WorkloadID: uuid.Generate().String(),
		IPAddress:  "172.16.0.2",
	}

	err = db.addInstance(&i)
	if err != nil {
		t.Fatal("unable to store instance")
	}

	instances, err := db.getInstances()
	if err != nil || len(instances) != 1 {
		t.Fatal(err)
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: "test",
	}

	err = db.createPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	m := types.MappedIP{
		ID:         uuid.Generate().String(),
		ExternalIP: "192.168.0.1",
		InternalIP: i.IPAddress,
		InstanceID: i.ID,
		TenantID:   i.TenantID,
		PoolID:     pool.ID,
		PoolName:   pool.Name,
	}

	err = db.createMappedIP(m)
	if err != nil {
		t.Fatal(err)
	}

	IPs := db.getMappedIPs()
	if len(IPs) != 1 {
		t.Fatal("could not get mapped IP")
	}

	if reflect.DeepEqual(IPs[m.ExternalIP], m) == false {
		t.Fatalf("expected %v, got %v\n", m, IPs[m.ExternalIP])
	}
}

func TestDeleteMappedIP(t *testing.T) {
	config := Config{
		PersistentURI: "file:deletedmappedaddress?mode=memory&cache=shared",
		TransientURI:  "file:deletemappedaddresst?mode=memory&cache=shared",
	}

	db, err := getPersistentStore(config)
	if err != nil {
		t.Fatal(err)
	}

	i := types.Instance{
		ID:         uuid.Generate().String(),
		TenantID:   uuid.Generate().String(),
		WorkloadID: uuid.Generate().String(),
		IPAddress:  "172.16.0.2",
	}

	err = db.addInstance(&i)
	if err != nil {
		t.Fatal("unable to store instance")
	}

	instances, err := db.getInstances()
	if err != nil || len(instances) != 1 {
		t.Fatal(err)
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: "test",
	}

	err = db.createPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	m := types.MappedIP{
		ID:         uuid.Generate().String(),
		ExternalIP: "192.168.0.1",
		InternalIP: i.IPAddress,
		InstanceID: i.ID,
		TenantID:   i.TenantID,
		PoolID:     pool.ID,
		PoolName:   pool.Name,
	}

	err = db.createMappedIP(m)
	if err != nil {
		t.Fatal(err)
	}

	IPs := db.getMappedIPs()
	if len(IPs) != 1 {
		t.Fatal("could not get mapped IP")
	}

	if reflect.DeepEqual(IPs[m.ExternalIP], m) == false {
		t.Fatalf("expected %v, got %v\n", m, IPs[m.ExternalIP])
	}

	err = db.deleteMappedIP(m.ID)
	if err != nil {
		t.Fatal(err)
	}

	IPs = db.getMappedIPs()
	if len(IPs) != 0 {
		t.Fatal("IP not deleted")
	}
}
