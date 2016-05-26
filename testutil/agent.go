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
	"fmt"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"gopkg.in/yaml.v2"
	"sync"
	"time"
)

// SsntpTestClient is global state for the testutil SSNTP client worker
type SsntpTestClient struct {
	Ssntp             ssntp.Client
	Name              string
	instances         []payloads.InstanceStat
	ticker            *time.Ticker
	UUID              string
	Role              ssntp.Role
	StartFail         bool
	StartFailReason   payloads.StartFailureReason
	StopFail          bool
	StopFailReason    payloads.StopFailureReason
	RestartFail       bool
	RestartFailReason payloads.RestartFailureReason
	traces            []*ssntp.Frame

	CmdChans     map[ssntp.Command]chan CmdResult
	CmdChansLock *sync.Mutex
}

// AddCmdChan opens and command channel to the SsntpTestClient
func (client *SsntpTestClient) AddCmdChan(cmd ssntp.Command, c chan CmdResult) {
	client.CmdChansLock.Lock()
	client.CmdChans[cmd] = c
	client.CmdChansLock.Unlock()
}

// ConnectNotify is an SSNTP callback stub for SsntpTestClient
func (client *SsntpTestClient) ConnectNotify() {
}

// DisconnectNotify is an SSNTP callback stub for SsntpTestClient
func (client *SsntpTestClient) DisconnectNotify() {
}

// StatusNotify is an SSNTP callback stub for SsntpTestClient
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

		client.instances = append(client.instances, istat)
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

// CommandNotify implements the SSNTP client CommandNotify callback for SsntpTestClient
func (client *SsntpTestClient) CommandNotify(command ssntp.Command, frame *ssntp.Frame) {
	payload := frame.Payload

	var result CmdResult

	if frame.Trace != nil {
		frame.SetEndStamp()
		client.traces = append(client.traces, frame)
	}

	client.CmdChansLock.Lock()
	c, ok := client.CmdChans[command]
	client.CmdChansLock.Unlock()

	switch command {
	case ssntp.START:
		result = client.handleStart(payload)

	case ssntp.STOP:
		result = client.handleStop(payload)

	case ssntp.RESTART:
		result = client.handleRestart(payload)
	}

	if ok {
		client.CmdChansLock.Lock()
		delete(client.CmdChans, command)
		client.CmdChansLock.Unlock()

		c <- result

		close(c)
	}

	return
}

// EventNotify implements the SSNTP client EventNotify callback for SsntpTestClient
func (client *SsntpTestClient) EventNotify(event ssntp.Event, frame *ssntp.Frame) {
}

// ErrorNotify implements the SSNTP client ErrorNotify callback for SsntpTestClient
func (client *SsntpTestClient) ErrorNotify(error ssntp.Error, frame *ssntp.Frame) {
}

// SendStats allows an SsntpTestClient to push an ssntp.STATS command frame
func (client *SsntpTestClient) SendStats() {
	stat := payloads.Stat{
		NodeUUID:        client.UUID,
		MemTotalMB:      256,
		MemAvailableMB:  256,
		DiskTotalMB:     1024,
		DiskAvailableMB: 1024,
		Load:            20,
		CpusOnline:      4,
		NodeHostName:    client.Name,
		Instances:       client.instances,
	}

	y, err := yaml.Marshal(stat)
	if err != nil {
		return
	}

	_, err = client.Ssntp.SendCommand(ssntp.STATS, y)
	if err != nil {
		fmt.Println(err)
	}
}

// SendTrace allows an SsntpTestClient to push an ssntp.TraceReport event frame
func (client *SsntpTestClient) SendTrace() {
	var s payloads.Trace

	for _, f := range client.traces {
		t, err := f.DumpTrace()
		if err != nil {
			fmt.Println(err)
			continue
		}

		s.Frames = append(s.Frames, *t)
	}

	payload, err := yaml.Marshal(&s)
	if err != nil {
		fmt.Println(err)
	}

	client.traces = nil

	_, err = client.Ssntp.SendEvent(ssntp.TraceReport, payload)
	if err != nil {
		fmt.Println(err)
	}
}

// SendDeleteEvent allows an SsntpTestClient to push an ssntp.InstanceDeleted event frame
func (client *SsntpTestClient) SendDeleteEvent(uuid string) {
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

	_, err = client.Ssntp.SendEvent(ssntp.InstanceDeleted, y)
	if err != nil {
		fmt.Println(err)
	}

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
