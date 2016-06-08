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
	"math/rand"
	"sync"
	"time"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"gopkg.in/yaml.v2"
)

// SsntpTestServer is global state for the testutil SSNTP server
type SsntpTestServer struct {
	Ssntp          ssntp.Server
	clients        []string
	clientsLock    *sync.Mutex
	CmdChans       map[ssntp.Command]chan CmdResult
	CmdChansLock   *sync.Mutex
	EventChans     map[ssntp.Event]chan CmdResult
	EventChansLock *sync.Mutex

	NetClients     map[string]bool
	NetClientsLock *sync.RWMutex
}

// AddCmdChan adds a command to the SsntpTestServer command channel
func (server *SsntpTestServer) AddCmdChan(cmd ssntp.Command) *chan CmdResult {
	c := make(chan CmdResult)

	server.CmdChansLock.Lock()
	server.CmdChans[cmd] = c
	server.CmdChansLock.Unlock()

	return &c
}

// GetCmdChanResult gets a CmdResult from the SsntpTestServer command channel
func (server *SsntpTestServer) GetCmdChanResult(c *chan CmdResult, cmd ssntp.Command) (result CmdResult, err error) {
	select {
	case result = <-*c:
		if result.Err != nil {
			err = fmt.Errorf("Server error on %s command: %s\n", cmd, result.Err)
		}
	case <-time.After(5 * time.Second):
		err = fmt.Errorf("Timeout waiting for server %s command result\n", cmd)
	}

	return result, err
}

// SendResultAndDelCmdChan deletes a command from the SsntpTestServer command channel
func (server *SsntpTestServer) SendResultAndDelCmdChan(cmd ssntp.Command, result CmdResult) {
	server.CmdChansLock.Lock()
	defer server.CmdChansLock.Unlock()
	c, ok := server.CmdChans[cmd]
	if ok {
		delete(server.CmdChans, cmd)
		c <- result
		close(c)
	}
}

// AddEventChan adds a command to the SsntpTestServer event channel
func (server *SsntpTestServer) AddEventChan(evt ssntp.Event) *chan CmdResult {
	c := make(chan CmdResult)

	server.EventChansLock.Lock()
	server.EventChans[evt] = c
	server.EventChansLock.Unlock()

	return &c
}

// GetEventChanResult gets a CmdResult from the SsntpTestServer event channel
func (server *SsntpTestServer) GetEventChanResult(c *chan CmdResult, evt ssntp.Event) (result CmdResult, err error) {
	select {
	case result = <-*c:
		if result.Err != nil {
			err = fmt.Errorf("Server error handling %s event: %s\n", evt, result.Err)
		}
	case <-time.After(5 * time.Second):
		err = fmt.Errorf("Timeout waiting for server %s event result\n", evt)
	}

	return result, err
}

// SendResultAndDelEventChan deletes an event from the SsntpTestServer event channel
func (server *SsntpTestServer) SendResultAndDelEventChan(evt ssntp.Event, result CmdResult) {
	server.EventChansLock.Lock()
	defer server.EventChansLock.Unlock()
	c, ok := server.EventChans[evt]
	if ok {
		delete(server.EventChans, evt)
		c <- result
		close(c)
	}
}

// ConnectNotify implements an SSNTP ConnectNotify callback for SsntpTestServer
func (server *SsntpTestServer) ConnectNotify(uuid string, role ssntp.Role) {
	var result CmdResult

	switch role {
	case ssntp.AGENT:
		server.clientsLock.Lock()
		defer server.clientsLock.Unlock()
		server.clients = append(server.clients, uuid)

	case ssntp.NETAGENT:
		server.NetClientsLock.Lock()
		server.NetClients[uuid] = true
		server.NetClientsLock.Unlock()
	}

	server.SendResultAndDelEventChan(ssntp.NodeConnected, result)
}

// DisconnectNotify implements an SSNTP DisconnectNotify callback for SsntpTestServer
func (server *SsntpTestServer) DisconnectNotify(uuid string, role ssntp.Role) {
	var result CmdResult

	server.clientsLock.Lock()
	for index := range server.clients {
		if server.clients[index] == uuid {
			server.clients = append(server.clients[:index], server.clients[index+1:]...)
			break
		}
	}
	server.clientsLock.Unlock()

	server.NetClientsLock.Lock()
	if server.NetClients[uuid] == true {
		delete(server.NetClients, uuid)
	}
	server.NetClientsLock.Unlock()

	server.SendResultAndDelEventChan(ssntp.NodeDisconnected, result)
}

// StatusNotify is an SSNTP callback stub for SsntpTestServer
func (server *SsntpTestServer) StatusNotify(uuid string, status ssntp.Status, frame *ssntp.Frame) {
}

// CommandNotify implements an SSNTP CommandNotify callback for SsntpTestServer
func (server *SsntpTestServer) CommandNotify(uuid string, command ssntp.Command, frame *ssntp.Frame) {
	var result CmdResult
	var nn bool

	payload := frame.Payload

	switch command {
	/*TODO:
	case CONNECT:
	case AssignPublicIP:
	case ReleasePublicIP:
	case CONFIGURE:
	*/
	case ssntp.START:
		var startCmd payloads.Start

		err := yaml.Unmarshal(payload, &startCmd)
		result.Err = err
		if err == nil {
			resources := startCmd.Start.RequestedResources

			for i := range resources {
				if resources[i].Type == payloads.NetworkNode {
					nn = true
					break
				}
			}
			result.InstanceUUID = startCmd.Start.InstanceUUID
			result.TenantUUID = startCmd.Start.TenantUUID
			result.CNCI = nn
		}

	case ssntp.DELETE:
		var delCmd payloads.Delete

		err := yaml.Unmarshal(payload, &delCmd)
		result.Err = err
		if err == nil {
			result.InstanceUUID = delCmd.Delete.InstanceUUID
			server.Ssntp.SendCommand(delCmd.Delete.WorkloadAgentUUID, command, frame.Payload)
		}

	case ssntp.STOP:
		var stopCmd payloads.Stop

		err := yaml.Unmarshal(payload, &stopCmd)
		result.Err = err
		if err == nil {
			result.InstanceUUID = stopCmd.Stop.InstanceUUID
			server.Ssntp.SendCommand(stopCmd.Stop.WorkloadAgentUUID, command, frame.Payload)
		}

	case ssntp.RESTART:
		var restartCmd payloads.Restart

		err := yaml.Unmarshal(payload, &restartCmd)
		result.Err = err
		if err == nil {
			result.InstanceUUID = restartCmd.Restart.InstanceUUID
			server.Ssntp.SendCommand(restartCmd.Restart.WorkloadAgentUUID, command, frame.Payload)
		}

	case ssntp.EVACUATE:
		var evacCmd payloads.Evacuate

		err := yaml.Unmarshal(payload, &evacCmd)
		result.Err = err
		if err == nil {
			result.NodeUUID = evacCmd.Evacuate.WorkloadAgentUUID
		}

	case ssntp.STATS:
		var statsCmd payloads.Stat

		err := yaml.Unmarshal(payload, &statsCmd)
		result.Err = err

	default:
		fmt.Printf("server unhandled command %s\n", command.String())
	}

	server.SendResultAndDelCmdChan(command, result)
}

// EventNotify implements an SSNTP EventNotify callback for SsntpTestServer
func (server *SsntpTestServer) EventNotify(uuid string, event ssntp.Event, frame *ssntp.Frame) {
}

// ErrorNotify is an SSNTP callback stub for SsntpTestServer
func (server *SsntpTestServer) ErrorNotify(uuid string, error ssntp.Error, frame *ssntp.Frame) {
}

// CommandForward implements an SSNTP CommandForward callback for SsntpTestServer
func (server *SsntpTestServer) CommandForward(uuid string, command ssntp.Command, frame *ssntp.Frame) (dest ssntp.ForwardDestination) {
	var startCmd payloads.Start
	var nn bool

	payload := frame.Payload

	err := yaml.Unmarshal(payload, &startCmd)

	if err != nil {
		return
	}

	resources := startCmd.Start.RequestedResources

	for i := range resources {
		if resources[i].Type == payloads.NetworkNode {
			nn = true
			break
		}
	}

	if nn {
		server.NetClientsLock.RLock()
		for key := range server.NetClients {
			dest.AddRecipient(key)
			break
		}
		server.NetClientsLock.RUnlock()
	} else if len(server.clients) > 0 {
		index := rand.Intn(len(server.clients))
		dest.AddRecipient(server.clients[index])
	}

	return dest
}

// StartTestServer starts a go routine for based on a
// testutil.SsntpTestServer configuration with standard ssntp.FrameRorwardRules
func StartTestServer(server *SsntpTestServer) {
	server.clientsLock = &sync.Mutex{}

	server.CmdChans = make(map[ssntp.Command]chan CmdResult)
	server.CmdChansLock = &sync.Mutex{}

	server.EventChans = make(map[ssntp.Event]chan CmdResult)
	server.EventChansLock = &sync.Mutex{}

	server.NetClients = make(map[string]bool)
	server.NetClientsLock = &sync.RWMutex{}

	serverConfig := ssntp.Config{
		CAcert: ssntp.DefaultCACert,
		Cert:   ssntp.RoleToDefaultCertName(ssntp.SERVER),
		Log:    ssntp.Log,
		ForwardRules: []ssntp.FrameForwardRule{
			{ // all STATS commands go to all Controllers
				Operand: ssntp.STATS,
				Dest:    ssntp.Controller,
			},
			{ // all TraceReport events go to all Controllers
				Operand: ssntp.TraceReport,
				Dest:    ssntp.Controller,
			},
			{ // all InstanceDeleted events go to all Controllers
				Operand: ssntp.InstanceDeleted,
				Dest:    ssntp.Controller,
			},
			{ // all ConcentratorInstanceAdded events go to all Controllers
				Operand: ssntp.ConcentratorInstanceAdded,
				Dest:    ssntp.Controller,
			},
			{ // all StartFailure events go to all Controllers
				Operand: ssntp.StartFailure,
				Dest:    ssntp.Controller,
			},
			{ // all StopFailure events go to all Controllers
				Operand: ssntp.StopFailure,
				Dest:    ssntp.Controller,
			},
			{ // all RestartFailure events go to all Controllers
				Operand: ssntp.RestartFailure,
				Dest:    ssntp.Controller,
			},
			{ // all START command are processed by the Command forwarder
				Operand:        ssntp.START,
				CommandForward: server,
			},
			{ // all RESTART command are processed by the Command forwarder
				Operand:        ssntp.RESTART,
				CommandForward: server,
			},
			{ // all STOP command are processed by the Command forwarder
				Operand:        ssntp.STOP,
				CommandForward: server,
			},
			{ // all DELETE command are processed by the Command forwarder
				Operand:        ssntp.DELETE,
				CommandForward: server,
			},
			{ // all EVACUATE command are processed by the Command forwarder
				Operand:        ssntp.EVACUATE,
				CommandForward: server,
			},
		},
	}

	go server.Ssntp.Serve(&serverConfig, server)
	return
}
