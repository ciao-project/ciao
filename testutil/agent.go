//
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

package testutil

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"gopkg.in/yaml.v2"
)

// SsntpTestClient is global state for the testutil SSNTP client worker
type SsntpTestClient struct {
	Ssntp             ssntp.Client
	Name              string
	instances         []payloads.InstanceStat
	instancesLock     *sync.Mutex
	ticker            *time.Ticker
	UUID              string
	Role              ssntp.Role
	StartFail         bool
	StartFailReason   payloads.StartFailureReason
	StopFail          bool
	StopFailReason    payloads.StopFailureReason
	RestartFail       bool
	RestartFailReason payloads.RestartFailureReason
	DeleteFail        bool
	DeleteFailReason  payloads.DeleteFailureReason
	traces            []*ssntp.Frame
	tracesLock        *sync.Mutex

	CmdChans       map[ssntp.Command]chan CmdResult
	CmdChansLock   *sync.Mutex
	EventChans     map[ssntp.Event]chan CmdResult
	EventChansLock *sync.Mutex
}

// NewSsntpTestClientConnection creates an SsntpTestClient and dials the server.
// Calling with a unique name parameter string for inclusion in the SsntpTestClient.Name
// field aides in debugging.  The role parameter is mandatory.  The uuid string
// parameter allows tests to specify a known uuid for simpler tests.
func NewSsntpTestClientConnection(name string, role ssntp.Role, uuid string) (*SsntpTestClient, error) {
	if role == ssntp.UNKNOWN {
		return nil, errors.New("no role specified")
	}
	if uuid == "" {
		return nil, errors.New("no uuid specified")
	}

	client := new(SsntpTestClient)
	client.Name = "Test " + role.String() + " " + name
	client.UUID = uuid
	client.Role = role
	client.CmdChans = make(map[ssntp.Command]chan CmdResult)
	client.CmdChansLock = &sync.Mutex{}
	client.EventChans = make(map[ssntp.Event]chan CmdResult)
	client.EventChansLock = &sync.Mutex{}
	client.instancesLock = &sync.Mutex{}
	client.tracesLock = &sync.Mutex{}

	config := &ssntp.Config{
		CAcert: ssntp.DefaultCACert,
		Cert:   ssntp.RoleToDefaultCertName(role),
		Log:    ssntp.Log,
		UUID:   client.UUID,
	}

	if err := client.Ssntp.Dial(config, client); err != nil {
		return nil, err
	}
	return client, nil
}

// AddCmdChan adds a command to the SsntpTestClient command channel
func (client *SsntpTestClient) AddCmdChan(cmd ssntp.Command) *chan CmdResult {
	c := make(chan CmdResult)

	client.CmdChansLock.Lock()
	client.CmdChans[cmd] = c
	client.CmdChansLock.Unlock()

	return &c
}

// GetCmdChanResult gets a CmdResult from the SsntpTestClient command channel
func (client *SsntpTestClient) GetCmdChanResult(c *chan CmdResult, cmd ssntp.Command) (result CmdResult, err error) {
	select {
	case result = <-*c:
		if result.Err != nil {
			err = fmt.Errorf("Client error sending %s command: %s\n", cmd, result.Err)
		}
	case <-time.After(5 * time.Second):
		err = fmt.Errorf("Timeout waiting for client %s command result\n", cmd)
	}

	return result, err
}

// SendResultAndDelCmdChan deletes a command from the SsntpTestClient command channel
func (client *SsntpTestClient) SendResultAndDelCmdChan(cmd ssntp.Command, result CmdResult) {
	client.CmdChansLock.Lock()
	defer client.CmdChansLock.Unlock()
	c, ok := client.CmdChans[cmd]
	if ok {
		delete(client.CmdChans, cmd)
		c <- result
		close(c)
	}
}

// AddEventChan adds a command to the SsntpTestClient event channel
func (client *SsntpTestClient) AddEventChan(evt ssntp.Event) *chan CmdResult {
	c := make(chan CmdResult)

	client.EventChansLock.Lock()
	client.EventChans[evt] = c
	client.EventChansLock.Unlock()

	return &c
}

// GetEventChanResult gets a CmdResult from the SsntpTestClient event channel
func (client *SsntpTestClient) GetEventChanResult(c *chan CmdResult, evt ssntp.Event) (result CmdResult, err error) {
	select {
	case result = <-*c:
		if result.Err != nil {
			err = fmt.Errorf("Client error sending %s event: %s\n", evt, result.Err)
		}
	case <-time.After(5 * time.Second):
		err = fmt.Errorf("Timeout waiting for client %s event result\n", evt)
	}

	return result, err
}

// SendResultAndDelEventChan deletes an event from the SsntpTestClient event channel
func (client *SsntpTestClient) SendResultAndDelEventChan(evt ssntp.Event, result CmdResult) {
	client.EventChansLock.Lock()
	defer client.EventChansLock.Unlock()
	c, ok := client.EventChans[evt]
	if ok {
		delete(client.EventChans, evt)
		c <- result
		close(c)
	}
}

// ConnectNotify implements the SSNTP client ConnectNotify callback for SsntpTestClient
func (client *SsntpTestClient) ConnectNotify() {
	var result CmdResult

	client.SendResultAndDelEventChan(ssntp.NodeConnected, result)
}

// DisconnectNotify implements the SSNTP client ConnectNotify callback for SsntpTestClient
func (client *SsntpTestClient) DisconnectNotify() {
	var result CmdResult

	client.SendResultAndDelEventChan(ssntp.NodeDisconnected, result)
}

// StatusNotify implements the SSNTP client StatusNotify callback for SsntpTestClient
func (client *SsntpTestClient) StatusNotify(status ssntp.Status, frame *ssntp.Frame) {
}

func (client *SsntpTestClient) handleStart(payload []byte) CmdResult {
	var result CmdResult
	var start payloads.Start

	err := yaml.Unmarshal(payload, &start)
	if err != nil {
		result.Err = err
		return result
	}

	result.InstanceUUID = start.Start.InstanceUUID
	result.TenantUUID = start.Start.TenantUUID
	result.NodeUUID = client.UUID

	if client.Role == ssntp.NETAGENT {
		networking := start.Start.Networking

		client.sendConcentratorAddedEvent(start.Start.InstanceUUID, start.Start.TenantUUID, networking.VnicMAC)
		result.CNCI = true
		return result
	}

	if !client.StartFail {
		istat := payloads.InstanceStat{
			InstanceUUID:  start.Start.InstanceUUID,
			State:         payloads.Running,
			MemoryUsageMB: 0,
			DiskUsageMB:   0,
			CPUUsage:      0,
		}

		client.instancesLock.Lock()
		client.instances = append(client.instances, istat)
		client.instancesLock.Unlock()
	} else {
		client.sendStartFailure(start.Start.InstanceUUID, client.StartFailReason)
	}

	return result
}

func (client *SsntpTestClient) handleStop(payload []byte) CmdResult {
	var result CmdResult
	var stopCmd payloads.Stop

	err := yaml.Unmarshal(payload, &stopCmd)
	if err != nil {
		result.Err = err
		return result
	}

	if !client.StopFail {
		client.instancesLock.Lock()
		defer client.instancesLock.Unlock()
		for i := range client.instances {
			istat := client.instances[i]
			if istat.InstanceUUID == stopCmd.Stop.InstanceUUID {
				client.instances[i].State = payloads.Exited
			}
		}
	} else {
		client.sendStopFailure(stopCmd.Stop.InstanceUUID, client.StopFailReason)
	}

	return result
}

func (client *SsntpTestClient) handleRestart(payload []byte) CmdResult {
	var result CmdResult
	var restartCmd payloads.Restart

	err := yaml.Unmarshal(payload, &restartCmd)
	if err != nil {
		result.Err = err
		return result
	}

	if !client.RestartFail {
		client.instancesLock.Lock()
		defer client.instancesLock.Unlock()
		for i := range client.instances {
			istat := client.instances[i]
			if istat.InstanceUUID == restartCmd.Restart.InstanceUUID {
				client.instances[i].State = payloads.Running
			}
		}
	} else {
		client.sendRestartFailure(restartCmd.Restart.InstanceUUID, client.RestartFailReason)
	}

	return result
}

func (client *SsntpTestClient) handleDelete(payload []byte) CmdResult {
	var result CmdResult
	var deleteCmd payloads.Delete

	err := yaml.Unmarshal(payload, &deleteCmd)
	if err != nil {
		result.Err = err
		return result
	}

	if !client.DeleteFail {
		client.instancesLock.Lock()
		defer client.instancesLock.Unlock()
		for i := range client.instances {
			istat := client.instances[i]
			if istat.InstanceUUID == deleteCmd.Delete.InstanceUUID {
				client.instances = append(client.instances[:i], client.instances[i+1:]...)
				break
			}
		}
	} else {
		client.sendDeleteFailure(deleteCmd.Delete.InstanceUUID, client.DeleteFailReason)
	}

	return result
}

// CommandNotify implements the SSNTP client CommandNotify callback for SsntpTestClient
func (client *SsntpTestClient) CommandNotify(command ssntp.Command, frame *ssntp.Frame) {
	payload := frame.Payload

	var result CmdResult

	if frame.Trace != nil {
		frame.SetEndStamp()
		client.tracesLock.Lock()
		client.traces = append(client.traces, frame)
		client.tracesLock.Unlock()
	}

	switch command {
	case ssntp.START:
		result = client.handleStart(payload)

	case ssntp.STOP:
		result = client.handleStop(payload)

	case ssntp.RESTART:
		result = client.handleRestart(payload)

	case ssntp.DELETE:
		result = client.handleDelete(payload)

	default:
		fmt.Printf("client unhandled command %s\n", command.String())
	}

	client.SendResultAndDelCmdChan(command, result)
}

// EventNotify is an SSNTP callback stub for SsntpTestClient
func (client *SsntpTestClient) EventNotify(event ssntp.Event, frame *ssntp.Frame) {
}

// ErrorNotify is an SSNTP callback stub for SsntpTestClient
func (client *SsntpTestClient) ErrorNotify(error ssntp.Error, frame *ssntp.Frame) {
}

// SendStats pushes an ssntp.STATS command frame from the SsntpTestClient
func (client *SsntpTestClient) SendStats() {
	var result CmdResult

	payload := StatsPayload(client.UUID, client.Name, client.instances, nil)

	y, err := yaml.Marshal(payload)
	if err != nil {
		result.Err = err
	} else {
		_, err = client.Ssntp.SendCommand(ssntp.STATS, y)
		if err != nil {
			result.Err = err
		}
	}

	client.CmdChansLock.Lock()
	defer client.CmdChansLock.Unlock()
	c, ok := client.CmdChans[ssntp.STATS]
	if ok {
		delete(client.CmdChans, ssntp.STATS)
		c <- result
		close(c)
	}
}

// SendTrace allows an SsntpTestClient to push an ssntp.TraceReport event frame
func (client *SsntpTestClient) SendTrace() {
	var result CmdResult
	var s payloads.Trace

	client.tracesLock.Lock()
	defer client.tracesLock.Unlock()

	for _, f := range client.traces {
		t, err := f.DumpTrace()
		if err != nil {
			continue
		}

		s.Frames = append(s.Frames, *t)
	}

	y, err := yaml.Marshal(&s)
	if err != nil {
		result.Err = err
	} else {
		client.traces = nil

		_, err = client.Ssntp.SendEvent(ssntp.TraceReport, y)
		if err != nil {
			result.Err = err
		}
	}

	client.SendResultAndDelEventChan(ssntp.TraceReport, result)
}

// SendDeleteEvent allows an SsntpTestClient to push an ssntp.InstanceDeleted event frame
func (client *SsntpTestClient) SendDeleteEvent(uuid string) {
	var result CmdResult

	evt := payloads.InstanceDeletedEvent{
		InstanceUUID: uuid,
	}

	event := payloads.EventInstanceDeleted{
		InstanceDeleted: evt,
	}

	y, err := yaml.Marshal(event)
	if err != nil {
		result.Err = err
	} else {
		_, err = client.Ssntp.SendEvent(ssntp.InstanceDeleted, y)
		if err != nil {
			result.Err = err
			fmt.Println(err)
		}
	}

	client.SendResultAndDelEventChan(ssntp.InstanceDeleted, result)
}

func (client *SsntpTestClient) sendConcentratorAddedEvent(instanceUUID string, tenantUUID string, vnicMAC string) {
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

	_, err = client.Ssntp.SendEvent(ssntp.ConcentratorInstanceAdded, y)
	if err != nil {
		fmt.Println(err)
	}
}

func (client *SsntpTestClient) sendStartFailure(instanceUUID string, reason payloads.StartFailureReason) {
	e := payloads.ErrorStartFailure{
		InstanceUUID: instanceUUID,
		Reason:       reason,
	}

	y, err := yaml.Marshal(e)
	if err != nil {
		return
	}

	_, err = client.Ssntp.SendError(ssntp.StartFailure, y)
	if err != nil {
		fmt.Println(err)
	}
}

func (client *SsntpTestClient) sendStopFailure(instanceUUID string, reason payloads.StopFailureReason) {
	e := payloads.ErrorStopFailure{
		InstanceUUID: instanceUUID,
		Reason:       reason,
	}

	y, err := yaml.Marshal(e)
	if err != nil {
		return
	}

	_, err = client.Ssntp.SendError(ssntp.StopFailure, y)
	if err != nil {
		fmt.Println(err)
	}
}

func (client *SsntpTestClient) sendRestartFailure(instanceUUID string, reason payloads.RestartFailureReason) {
	e := payloads.ErrorRestartFailure{
		InstanceUUID: instanceUUID,
		Reason:       reason,
	}

	y, err := yaml.Marshal(e)
	if err != nil {
		return
	}

	_, err = client.Ssntp.SendError(ssntp.RestartFailure, y)
	if err != nil {
		fmt.Println(err)
	}
}

func (client *SsntpTestClient) sendDeleteFailure(instanceUUID string, reason payloads.DeleteFailureReason) {
	e := payloads.ErrorDeleteFailure{
		InstanceUUID: instanceUUID,
		Reason:       reason,
	}

	y, err := yaml.Marshal(e)
	if err != nil {
		return
	}

	_, err = client.Ssntp.SendError(ssntp.DeleteFailure, y)
	if err != nil {
		fmt.Println(err)
	}
}
