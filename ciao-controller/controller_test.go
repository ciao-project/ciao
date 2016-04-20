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
	"flag"
	"fmt"
	datastore "github.com/01org/ciao/ciao-controller/internal/datastore"
	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/docker/distribution/uuid"
	"gopkg.in/yaml.v2"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

type ssntpTestServer struct {
	ssntp        ssntp.Server
	clients      []string
	cmdChans     map[ssntp.Command]chan cmdResult
	cmdChansLock *sync.Mutex

	netClients     map[string]bool
	netClientsLock *sync.RWMutex
}

type cmdResult struct {
	instanceUUID string
	err          error
	nodeUUID     string
	tenantUUID   string
	cnci         bool
}

func (server *ssntpTestServer) addCmdChan(cmd ssntp.Command, c chan cmdResult) {
	server.cmdChansLock.Lock()
	server.cmdChans[cmd] = c
	server.cmdChansLock.Unlock()
}

func (server *ssntpTestServer) ConnectNotify(uuid string, role uint32) {
	switch role {
	case ssntp.AGENT:
		server.clients = append(server.clients, uuid)

	case ssntp.NETAGENT:
		server.netClientsLock.Lock()
		server.netClients[uuid] = true
		server.netClientsLock.Unlock()
	}

}

func (server *ssntpTestServer) DisconnectNotify(uuid string) {
	for index := range server.clients {
		if server.clients[index] == uuid {
			server.clients = append(server.clients[:index], server.clients[index+1:]...)
			return
		}
	}

	server.netClientsLock.Lock()
	delete(server.netClients, uuid)
	server.netClientsLock.Unlock()
}

func (server *ssntpTestServer) StatusNotify(uuid string, status ssntp.Status, frame *ssntp.Frame) {
}

func (server *ssntpTestServer) CommandNotify(uuid string, command ssntp.Command, frame *ssntp.Frame) {
	var result cmdResult
	var nn bool

	payload := frame.Payload

	server.cmdChansLock.Lock()
	c, ok := server.cmdChans[command]
	server.cmdChansLock.Unlock()

	switch command {
	case ssntp.START:
		var startCmd payloads.Start

		err := yaml.Unmarshal(payload, &startCmd)

		if err == nil {
			resources := startCmd.Start.RequestedResources

			for i := range resources {
				if resources[i].Type == payloads.NetworkNode {
					nn = true
					break
				}
			}

			if nn {
				server.netClientsLock.RLock()
				for key := range server.netClients {
					server.ssntp.SendCommand(key, command, frame.Payload)
					break
				}
				server.netClientsLock.RUnlock()
			} else if len(server.clients) > 0 {
				index := rand.Intn(len(server.clients))
				server.ssntp.SendCommand(server.clients[index], command, frame.Payload)
			}
		}

		if ok {
			if err != nil {
				result.err = err
			} else {
				result.instanceUUID = startCmd.Start.InstanceUUID
				result.tenantUUID = startCmd.Start.TenantUUID
				result.cnci = nn
			}

		}

	case ssntp.DELETE:
		if ok {
			var delCmd payloads.Delete

			err := yaml.Unmarshal(payload, &delCmd)
			if err != nil {
				result.err = err
			} else {
				result.instanceUUID = delCmd.Delete.InstanceUUID
			}
		}

	case ssntp.STOP:
		if ok {
			var stopCmd payloads.Stop

			err := yaml.Unmarshal(payload, &stopCmd)
			if err != nil {
				result.err = err
			} else {
				result.instanceUUID = stopCmd.Stop.InstanceUUID
				server.ssntp.SendCommand(stopCmd.Stop.WorkloadAgentUUID, command, frame.Payload)
			}
		}

	case ssntp.RESTART:
		if ok {
			var restartCmd payloads.Restart

			err := yaml.Unmarshal(payload, &restartCmd)
			if err != nil {
				result.err = err
			} else {
				result.instanceUUID = restartCmd.Restart.InstanceUUID
				server.ssntp.SendCommand(restartCmd.Restart.WorkloadAgentUUID, command, frame.Payload)
			}
		}

	case ssntp.EVACUATE:
		if ok {
			var evacCmd payloads.Evacuate

			err := yaml.Unmarshal(payload, &evacCmd)
			if err != nil {
				result.err = err
			} else {
				result.nodeUUID = evacCmd.Evacuate.WorkloadAgentUUID
			}
		}
	}

	if ok {
		server.cmdChansLock.Lock()
		delete(server.cmdChans, command)
		server.cmdChansLock.Unlock()

		c <- result

		close(c)
	}
}

func (server *ssntpTestServer) EventNotify(uuid string, event ssntp.Event, frame *ssntp.Frame) {
}

func (server *ssntpTestServer) ErrorNotify(uuid string, error ssntp.Error, frame *ssntp.Frame) {
}

type ssntpTestClient struct {
	ssntp             ssntp.Client
	name              string
	instances         []payloads.InstanceStat
	ticker            *time.Ticker
	uuid              string
	role              ssntp.Role
	startFail         bool
	startFailReason   payloads.StartFailureReason
	stopFail          bool
	stopFailReason    payloads.StopFailureReason
	restartFail       bool
	restartFailReason payloads.RestartFailureReason
}

func (client *ssntpTestClient) ConnectNotify() {
}

func (client *ssntpTestClient) DisconnectNotify() {
}

func (client *ssntpTestClient) StatusNotify(status ssntp.Status, frame *ssntp.Frame) {
}

func (client *ssntpTestClient) CommandNotify(command ssntp.Command, frame *ssntp.Frame) {
	var start payloads.Start
	payload := frame.Payload

	switch command {
	case ssntp.START:
		err := yaml.Unmarshal(payload, &start)
		if err != nil {
			return
		}

		if client.role == ssntp.NETAGENT {
			networking := start.Start.Networking

			client.sendConcentratorAddedEvent(start.Start.InstanceUUID, start.Start.TenantUUID, networking.VnicMAC)
			return
		}

		if !client.startFail {
			istat := payloads.InstanceStat{
				InstanceUUID:  start.Start.InstanceUUID,
				State:         payloads.Running,
				MemoryUsageMB: 0,
				DiskUsageMB:   0,
				CPUUsage:      0,
			}

			client.instances = append(client.instances, istat)
		} else {
			client.sendStartFailure(start.Start.InstanceUUID, client.startFailReason)
		}

	case ssntp.STOP:
		var stopCmd payloads.Stop

		err := yaml.Unmarshal(payload, &stopCmd)
		if err != nil {
			return
		}

		if !client.stopFail {
			for i := range client.instances {
				istat := client.instances[i]
				if istat.InstanceUUID == stopCmd.Stop.InstanceUUID {
					client.instances[i].State = payloads.Exited
				}
			}
		} else {
			client.sendStopFailure(stopCmd.Stop.InstanceUUID, client.stopFailReason)
		}

	case ssntp.RESTART:
		var restartCmd payloads.Restart

		err := yaml.Unmarshal(payload, &restartCmd)
		if err != nil {
			return
		}

		if !client.restartFail {
			for i := range client.instances {
				istat := client.instances[i]
				if istat.InstanceUUID == restartCmd.Restart.InstanceUUID {
					client.instances[i].State = payloads.Running
				}
			}
		} else {
			client.sendRestartFailure(restartCmd.Restart.InstanceUUID, client.restartFailReason)
		}
	}
}

func (client *ssntpTestClient) EventNotify(event ssntp.Event, frame *ssntp.Frame) {
}

func (client *ssntpTestClient) ErrorNotify(error ssntp.Error, frame *ssntp.Frame) {
}

func newTestClient(num int, role ssntp.Role) *ssntpTestClient {
	client := &ssntpTestClient{
		name: "Test " + role.String() + strconv.Itoa(num),
		uuid: uuid.Generate().String(),
		role: role,
	}

	config := &ssntp.Config{
		Role:   uint32(role),
		CAcert: *caCert,
		Cert:   *cert,
		Log:    ssntp.Log,
		UUID:   client.uuid,
	}

	if client.ssntp.Dial(config, client) != nil {
		return nil
	}

	return client
}

func (client *ssntpTestClient) sendStats() {
	stat := payloads.Stat{
		NodeUUID:        client.uuid,
		MemTotalMB:      256,
		MemAvailableMB:  256,
		DiskTotalMB:     1024,
		DiskAvailableMB: 1024,
		Load:            20,
		CpusOnline:      4,
		NodeHostName:    client.name,
		Instances:       client.instances,
	}

	y, err := yaml.Marshal(stat)
	if err != nil {
		return
	}

	_, err = client.ssntp.SendCommand(ssntp.STATS, y)
	if err != nil {
		fmt.Println(err)
	}
}

func (client *ssntpTestClient) sendDeleteEvent(uuid string) {
	evt := payloads.InstanceDeletedEvent{
		InstanceUUID: uuid,
	}

	event := payloads.EventInstanceDeleted{
		InstanceDeleted: evt,
	}

	y, err := yaml.Marshal(event)
	if err != nil {
		return
	}

	_, err = client.ssntp.SendEvent(ssntp.InstanceDeleted, y)
	if err != nil {
		fmt.Println(err)
	}

}

func (client *ssntpTestClient) sendConcentratorAddedEvent(instanceUUID string, tenantUUID string, vnicMAC string) {
	evt := payloads.ConcentratorInstanceAddedEvent{
		InstanceUUID:    instanceUUID,
		TenantUUID:      tenantUUID,
		ConcentratorIP:  "192.168.0.1",
		ConcentratorMAC: vnicMAC,
	}

	event := payloads.EventConcentratorInstanceAdded{
		CNCIAdded: evt,
	}

	y, err := yaml.Marshal(event)
	if err != nil {
		return
	}

	_, err = client.ssntp.SendEvent(ssntp.ConcentratorInstanceAdded, y)
	if err != nil {
		fmt.Println(err)
	}
}

func (client *ssntpTestClient) sendStartFailure(instanceUUID string, reason payloads.StartFailureReason) {
	e := payloads.ErrorStartFailure{
		InstanceUUID: instanceUUID,
		Reason:       reason,
	}

	y, err := yaml.Marshal(e)
	if err != nil {
		return
	}

	_, err = client.ssntp.SendError(ssntp.StartFailure, y)
	if err != nil {
		fmt.Println(err)
	}
}

func (client *ssntpTestClient) sendStopFailure(instanceUUID string, reason payloads.StopFailureReason) {
	e := payloads.ErrorStopFailure{
		InstanceUUID: instanceUUID,
		Reason:       reason,
	}

	y, err := yaml.Marshal(e)
	if err != nil {
		return
	}

	_, err = client.ssntp.SendError(ssntp.StopFailure, y)
	if err != nil {
		fmt.Println(err)
	}
}

func (client *ssntpTestClient) sendRestartFailure(instanceUUID string, reason payloads.RestartFailureReason) {
	e := payloads.ErrorRestartFailure{
		InstanceUUID: instanceUUID,
		Reason:       reason,
	}

	y, err := yaml.Marshal(e)
	if err != nil {
		return
	}

	_, err = client.ssntp.SendError(ssntp.RestartFailure, y)
	if err != nil {
		fmt.Println(err)
	}
}

func startTestServer(server *ssntpTestServer) {
	server.cmdChans = make(map[ssntp.Command]chan cmdResult)
	server.cmdChansLock = &sync.Mutex{}

	server.netClients = make(map[string]bool)
	server.netClientsLock = &sync.RWMutex{}

	serverConfig := ssntp.Config{
		Role:   ssntp.SERVER,
		CAcert: *caCert,
		Cert:   *cert,
		Log:    ssntp.Log,
		ForwardRules: []ssntp.FrameForwardRule{
			{
				Operand: ssntp.STATS,
				Dest:    ssntp.Controller,
			},
			{
				Operand: ssntp.InstanceDeleted,
				Dest:    ssntp.Controller,
			},
			{
				Operand: ssntp.ConcentratorInstanceAdded,
				Dest:    ssntp.Controller,
			},
			{
				Operand: ssntp.StartFailure,
				Dest:    ssntp.Controller,
			},
			{
				Operand: ssntp.StopFailure,
				Dest:    ssntp.Controller,
			},
			{
				Operand: ssntp.RestartFailure,
				Dest:    ssntp.Controller,
			},
		},
	}

	go server.ssntp.Serve(&serverConfig, server)
	return
}

func addTestTenant() (tenant *types.Tenant, err error) {
	/* add a new tenant */
	tuuid := uuid.Generate()
	tenant, err = context.ds.AddTenant(tuuid.String())
	if err != nil {
		return
	}

	// Add fake CNCI
	err = context.ds.AddTenantCNCI(tuuid.String(), uuid.Generate().String(), tenant.CNCIMAC)
	if err != nil {
		return
	}
	err = context.ds.AddCNCIIP(tenant.CNCIMAC, "192.168.0.1")
	if err != nil {
		return
	}
	return
}

func BenchmarkStartSingleWorkload(b *testing.B) {
	var err error

	/* add a new tenant */
	tuuid := uuid.Generate()
	tenant, err := context.ds.AddTenant(tuuid.String())
	if err != nil {
		b.Error(err)
	}

	// Add fake CNCI
	err = context.ds.AddTenantCNCI(tuuid.String(), uuid.Generate().String(), tenant.CNCIMAC)
	if err != nil {
		b.Error(err)
	}
	err = context.ds.AddCNCIIP(tenant.CNCIMAC, "192.168.0.1")
	if err != nil {
		b.Error(err)
	}

	// get workload ID
	wls, err := context.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		b.Fatal(err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err = context.startWorkload(wls[0].ID, tuuid.String(), 1, false, "")
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkStart1000Workload(b *testing.B) {
	var err error

	/* add a new tenant */
	tuuid := uuid.Generate()
	tenant, err := context.ds.AddTenant(tuuid.String())
	if err != nil {
		b.Error(err)
	}

	// Add fake CNCI
	err = context.ds.AddTenantCNCI(tuuid.String(), uuid.Generate().String(), tenant.CNCIMAC)
	if err != nil {
		b.Error(err)
	}
	err = context.ds.AddCNCIIP(tenant.CNCIMAC, "192.168.0.1")
	if err != nil {
		b.Error(err)
	}

	// get workload ID
	wls, err := context.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		b.Fatal(err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err = context.startWorkload(wls[0].ID, tuuid.String(), 1000, false, "")
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkNewConfig(b *testing.B) {
	var err error

	tenant, err := addTestTenant()
	if err != nil {
		b.Error(err)
	}

	// get workload ID
	wls, err := context.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		b.Fatal(err)
	}

	id := uuid.Generate()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := newConfig(context, wls[0], id.String(), tenant.ID)
		if err != nil {
			b.Error(err)
		}
	}
}

func TestTenantWithinBounds(t *testing.T) {
	var err error

	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	/* put tenant limit of 1 instance */
	err = context.ds.AddLimit(tenant.ID, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	wls, err := context.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		t.Fatal(err)
	}

	_, err = context.startWorkload(wls[0].ID, tenant.ID, 1, false, "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestTenantOutOfBounds(t *testing.T) {
	var err error

	/* add a new tenant */
	tenant, err := addTestTenant()
	if err != nil {
		t.Error(err)
	}

	/* put tenant limit of 1 instance */
	_ = context.ds.AddLimit(tenant.ID, 1, 1)

	wls, err := context.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		t.Fatal(err)
	}

	/* try to send 2 workload start commands */
	_, err = context.startWorkload(wls[0].ID, tenant.ID, 2, false, "")
	if err == nil {
		t.Errorf("Not tracking limits correctly")
	}
}

// TestNewTenantHardwareAddr
// Confirm that the mac addresses generated from a given
// IP address is as expected.
func TestNewTenantHardwareAddr(t *testing.T) {
	ip := net.ParseIP("172.16.0.2")
	expectedMAC := "02:00:ac:10:00:02"
	hw := newTenantHardwareAddr(ip)
	if hw.String() != expectedMAC {
		t.Error("Expected: ", expectedMAC, " Received: ", hw.String())
	}
}

func TestStartWorkload(t *testing.T) {
	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	wls, err := context.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		t.Fatal(err)
	}

	c := make(chan cmdResult)
	server.addCmdChan(ssntp.START, c)

	instances, err := context.startWorkload(wls[0].ID, tenant.ID, 1, false, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(instances) != 1 {
		t.Error(err)
	}

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.instanceUUID != instances[0].ID {
			t.Fatal("Did not get correct Instance ID")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for START command")
	}
}

func TestStartWorkloadLaunchCNCI(t *testing.T) {
	netClient := newTestClient(0, ssntp.NETAGENT)

	wls, err := context.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		t.Fatal(err)
	}

	c := make(chan cmdResult)
	server.addCmdChan(ssntp.START, c)

	id := uuid.Generate().String()

	var instances []*types.Instance

	go func() {
		instances, err = context.startWorkload(wls[0].ID, id, 1, false, "")
		if err != nil {
			t.Fatal(err)
		}

		if len(instances) != 1 {
			t.Fatal(err)
		}
	}()

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.tenantUUID != id {
			t.Fatal("Did not get correct tenant ID")
		}

		if !result.cnci {
			t.Fatal("this is not a CNCI launch request")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for START command for CNCI")
	}

	c = make(chan cmdResult)
	server.addCmdChan(ssntp.START, c)

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.instanceUUID != instances[0].ID {
			t.Fatal("Did not get correct Instance ID")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for START command")
	}

	time.Sleep(1 * time.Second)

	tenant, err := context.ds.GetTenant(id)
	if err != nil {
		t.Fatal(err)
	}

	if tenant.CNCIIP == "" {
		t.Fatal("CNCI Info not updated")
	}

	netClient.ssntp.Close()
}

// TBD: for the launch CNCI tests, I really need to create a fake
// network node and test that way.

func TestDeleteInstance(t *testing.T) {
	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	wls, err := context.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		t.Fatal(err)
	}

	client := newTestClient(0, ssntp.AGENT)

	c := make(chan cmdResult)
	server.addCmdChan(ssntp.START, c)

	instances, err := context.startWorkload(wls[0].ID, tenant.ID, 1, false, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(instances) != 1 {
		t.Error(err)
	}

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.instanceUUID != instances[0].ID {
			t.Fatal("Did not get correct Instance ID")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for START command")
	}

	time.Sleep(1 * time.Second)

	client.sendStats()

	c = make(chan cmdResult)
	server.addCmdChan(ssntp.DELETE, c)

	time.Sleep(1 * time.Second)

	err = context.deleteInstance(instances[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.instanceUUID != instances[0].ID {
			t.Fatal("Did not get correct Instance ID")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for DELETE command")
	}

	client.ssntp.Close()
}

func TestStopInstance(t *testing.T) {
	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	wls, err := context.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		t.Fatal(err)
	}

	client := newTestClient(0, ssntp.AGENT)

	c := make(chan cmdResult)
	server.addCmdChan(ssntp.START, c)

	instances, err := context.startWorkload(wls[0].ID, tenant.ID, 1, false, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(instances) != 1 {
		t.Error(err)
	}

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.instanceUUID != instances[0].ID {
			t.Fatal("Did not get correct Instance ID")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for START command")
	}

	time.Sleep(1 * time.Second)

	client.sendStats()

	c = make(chan cmdResult)
	server.addCmdChan(ssntp.STOP, c)

	time.Sleep(1 * time.Second)

	err = context.stopInstance(instances[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.instanceUUID != instances[0].ID {
			t.Fatal("Did not get correct Instance ID")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for STOP command")
	}

	client.ssntp.Close()
}

func TestRestartInstance(t *testing.T) {
	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	wls, err := context.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		t.Fatal(err)
	}

	client := newTestClient(0, ssntp.AGENT)

	c := make(chan cmdResult)
	server.addCmdChan(ssntp.START, c)

	instances, err := context.startWorkload(wls[0].ID, tenant.ID, 1, false, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(instances) != 1 {
		t.Error(err)
	}

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.instanceUUID != instances[0].ID {
			t.Fatal("Did not get correct Instance ID")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for START command")
	}

	time.Sleep(1 * time.Second)

	client.sendStats()

	c = make(chan cmdResult)
	server.addCmdChan(ssntp.STOP, c)

	time.Sleep(1 * time.Second)

	err = context.stopInstance(instances[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.instanceUUID != instances[0].ID {
			t.Fatal("Did not get correct Instance ID")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for STOP command")
	}

	// now attempt to restart
	time.Sleep(1 * time.Second)

	client.sendStats()

	c = make(chan cmdResult)
	server.addCmdChan(ssntp.RESTART, c)

	time.Sleep(1 * time.Second)

	err = context.restartInstance(instances[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.instanceUUID != instances[0].ID {
			t.Fatal("Did not get correct Instance ID")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for RESTART command")
	}

	client.ssntp.Close()
}

func TestEvacuateNode(t *testing.T) {
	client := newTestClient(0, ssntp.AGENT)

	c := make(chan cmdResult)
	server.addCmdChan(ssntp.EVACUATE, c)

	// ok to not send workload first?

	err := context.evacuateNode(client.uuid)
	if err != nil {
		t.Error(err)
	}

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.nodeUUID != client.uuid {
			t.Fatal("Did not get node ID")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for EVACUATE command")
	}

	client.ssntp.Close()
}

func TestInstanceDeletedEvent(t *testing.T) {
	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	wls, err := context.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		t.Fatal(err)
	}

	client := newTestClient(0, ssntp.AGENT)

	instances, err := context.startWorkload(wls[0].ID, tenant.ID, 1, false, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(instances) != 1 {
		t.Error(err)
	}

	time.Sleep(1 * time.Second)

	client.sendStats()

	time.Sleep(1 * time.Second)

	// right now I don't have this forwarded to the client
	// so this step is probably not necessary
	err = context.deleteInstance(instances[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)

	client.sendDeleteEvent(instances[0].ID)

	time.Sleep(1 * time.Second)

	// try to get instance info
	_, err = context.ds.GetInstance(instances[0].ID)
	if err == nil {
		t.Error("Instance not deleted")
	}

	client.ssntp.Close()
}

func TestLaunchCNCI(t *testing.T) {
	netClient := newTestClient(0, ssntp.NETAGENT)

	c := make(chan cmdResult)
	server.addCmdChan(ssntp.START, c)

	id := uuid.Generate().String()

	// this blocks till it get success or failure
	go context.addTenant(id)

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.tenantUUID != id {
			t.Fatal("Did not get correct tenant ID")
		}

		if !result.cnci {
			t.Fatal("this is not a CNCI launch request")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for START command")
	}

	time.Sleep(2 * time.Second)

	tenant, err := context.ds.GetTenant(id)
	if err != nil || tenant == nil {
		t.Fatal(err)
	}

	if tenant.CNCIIP == "" {
		t.Fatal("CNCI Info not updated")
	}

	netClient.ssntp.Close()
}

func TestStartFailure(t *testing.T) {
	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	wls, err := context.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		t.Fatal(err)
	}

	client := newTestClient(0, ssntp.AGENT)
	client.startFail = true
	client.startFailReason = payloads.FullCloud

	c := make(chan cmdResult)
	server.addCmdChan(ssntp.START, c)

	instances, err := context.startWorkload(wls[0].ID, tenant.ID, 1, false, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(instances) != 1 {
		t.Error(err)
	}

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.instanceUUID != instances[0].ID {
			t.Fatal("Did not get correct Instance ID")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for START command")
	}

	// since we had a start failure, we should confirm that the
	// instance is no longer pending in the database
	client.ssntp.Close()
}

func TestStopFailure(t *testing.T) {
	context.ds.ClearLog()

	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	wls, err := context.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		t.Fatal(err)
	}

	client := newTestClient(0, ssntp.AGENT)
	client.stopFail = true
	client.stopFailReason = payloads.StopNoInstance

	c := make(chan cmdResult)
	server.addCmdChan(ssntp.START, c)

	instances, err := context.startWorkload(wls[0].ID, tenant.ID, 1, false, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(instances) != 1 {
		t.Error(err)
	}

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.instanceUUID != instances[0].ID {
			t.Fatal("Did not get correct Instance ID")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for START command")
	}

	time.Sleep(1 * time.Second)

	client.sendStats()

	time.Sleep(1 * time.Second)

	c = make(chan cmdResult)
	server.addCmdChan(ssntp.STOP, c)

	time.Sleep(1 * time.Second)

	err = context.stopInstance(instances[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.instanceUUID != instances[0].ID {
			t.Fatal("Did not get correct Instance ID")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for STOP command")
	}

	time.Sleep(1 * time.Second)

	client.ssntp.Close()

	// the response to a stop failure is to log the failure
	entries, err := context.ds.GetEventLog()
	if err != nil {
		t.Fatal(err)
	}

	expectedMsg := fmt.Sprintf("Stop Failure %s: %s", instances[0].ID, client.stopFailReason.String())

	for i := range entries {
		if entries[i].Message == expectedMsg {
			return
		}
	}
	t.Error("Did not find failure message in Log")
}

func TestRestartFailure(t *testing.T) {
	context.ds.ClearLog()

	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	wls, err := context.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		t.Fatal(err)
	}

	client := newTestClient(0, ssntp.AGENT)
	client.restartFail = true
	client.restartFailReason = payloads.RestartLaunchFailure

	c := make(chan cmdResult)
	server.addCmdChan(ssntp.START, c)

	instances, err := context.startWorkload(wls[0].ID, tenant.ID, 1, false, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(instances) != 1 {
		t.Error(err)
	}

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.instanceUUID != instances[0].ID {
			t.Fatal("Did not get correct Instance ID")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for START command")
	}

	time.Sleep(1 * time.Second)

	client.sendStats()

	time.Sleep(1 * time.Second)

	c = make(chan cmdResult)
	server.addCmdChan(ssntp.STOP, c)

	err = context.stopInstance(instances[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.instanceUUID != instances[0].ID {
			t.Fatal("Did not get correct Instance ID")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for STOP command")
	}

	time.Sleep(1 * time.Second)

	client.sendStats()

	time.Sleep(1 * time.Second)

	c = make(chan cmdResult)
	server.addCmdChan(ssntp.RESTART, c)

	err = context.restartInstance(instances[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.instanceUUID != instances[0].ID {
			t.Fatal("Did not get correct Instance ID")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for RESTART command")
	}

	time.Sleep(1 * time.Second)

	client.ssntp.Close()

	// the response to a restart failure is to log the failure
	entries, err := context.ds.GetEventLog()
	if err != nil {
		t.Fatal(err)
	}

	expectedMsg := fmt.Sprintf("Restart Failure %s: %s", instances[0].ID, client.restartFailReason.String())

	for i := range entries {
		if entries[i].Message == expectedMsg {
			return
		}
	}
	t.Error("Did not find failure message in Log")
}

func TestNoNetwork(t *testing.T) {
	nn := true

	noNetwork = &nn

	id := uuid.Generate().String()

	wls, err := context.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		t.Fatal(err)
	}

	c := make(chan cmdResult)
	server.addCmdChan(ssntp.START, c)

	instances, err := context.startWorkload(wls[0].ID, id, 1, false, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(instances) != 1 {
		t.Error(err)
	}

	select {
	case result := <-c:
		if result.err != nil {
			t.Fatal("Error parsing command yaml")
		}

		if result.instanceUUID != instances[0].ID {
			t.Fatal("Did not get correct Instance ID")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for START command")
	}
}

var testClients []*ssntpTestClient
var context *controller
var server ssntpTestServer

func TestMain(m *testing.M) {
	flag.Parse()

	// create fake ssntp server
	startTestServer(&server)
	defer server.ssntp.Stop()

	context = new(controller)
	context.ds = new(datastore.Datastore)

	dsConfig := datastore.Config{
		PersistentURI:     "./ciao-controller-test.db",
		TransientURI:      "./ciao-controller-test-tdb.db",
		InitTablesPath:    *tablesInitPath,
		InitWorkloadsPath: *workloadsPath,
	}

	err := context.ds.Init(dsConfig)
	if err != nil {
		os.Exit(1)
	}

	config := &ssntp.Config{
		URI:    "localhost",
		CAcert: *caCert,
		Cert:   *cert,
		Role:   ssntp.Controller,
	}

	context.client, err = newSSNTPClient(context, config)
	if err != nil {
		os.Exit(1)
	}

	code := m.Run()

	context.client.Disconnect()
	context.ds.Exit()

	os.Remove("./ciao-controller-test.db")
	os.Remove("./ciao-controller-test.db-shm")
	os.Remove("./ciao-controller-test.db-wal")
	os.Remove("./ciao-controller-test-tdb.db")
	os.Remove("./ciao-controller-test-tdb.db-shm")
	os.Remove("./ciao-controller-test-tdb.db-wal")

	os.Exit(code)
}
