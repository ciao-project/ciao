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
	"math/rand"
	"sync"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"gopkg.in/yaml.v2"
)

// SsntpTestServer is global state for the testutil SSNTP server
type SsntpTestServer struct {
	Ssntp        ssntp.Server
	clients      []string
	CmdChans     map[ssntp.Command]chan CmdResult
	CmdChansLock *sync.Mutex

	NetClients     map[string]bool
	NetClientsLock *sync.RWMutex
}

// AddCmdChan opens and command channel to the SsntpTestServer
func (server *SsntpTestServer) AddCmdChan(cmd ssntp.Command, c chan CmdResult) {
	server.CmdChansLock.Lock()
	server.CmdChans[cmd] = c
	server.CmdChansLock.Unlock()
}

// ConnectNotify implements an SSNTP ConnectNotify callback for SsntpTestServer
func (server *SsntpTestServer) ConnectNotify(uuid string, role ssntp.Role) {
	switch role {
	case ssntp.AGENT:
		server.clients = append(server.clients, uuid)

	case ssntp.NETAGENT:
		server.NetClientsLock.Lock()
		server.NetClients[uuid] = true
		server.NetClientsLock.Unlock()
	}

}

// DisconnectNotify implements an SSNTP DisconnectNotify callback for SsntpTestServer
func (server *SsntpTestServer) DisconnectNotify(uuid string, role ssntp.Role) {
	for index := range server.clients {
		if server.clients[index] == uuid {
			server.clients = append(server.clients[:index], server.clients[index+1:]...)
			return
		}
	}

	server.NetClientsLock.Lock()
	delete(server.NetClients, uuid)
	server.NetClientsLock.Unlock()
}

// StatusNotify is an SSNTP callback stub for SsntpTestServer
func (server *SsntpTestServer) StatusNotify(uuid string, status ssntp.Status, frame *ssntp.Frame) {
}

// CommandNotify implements an SSNTP CommandNotify callback for SsntpTestServer
func (server *SsntpTestServer) CommandNotify(uuid string, command ssntp.Command, frame *ssntp.Frame) {
	var result CmdResult
	var nn bool

	payload := frame.Payload

	server.CmdChansLock.Lock()
	c, ok := server.CmdChans[command]
	server.CmdChansLock.Unlock()

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
			result.InstanceUUID = startCmd.Start.InstanceUUID
			result.TenantUUID = startCmd.Start.TenantUUID
			result.CNCI = nn
		}
		result.Err = err

	case ssntp.DELETE:
		var delCmd payloads.Delete

		err := yaml.Unmarshal(payload, &delCmd)
		result.Err = err
		if err == nil {
			result.InstanceUUID = delCmd.Delete.InstanceUUID
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
	}

	if ok {
		server.CmdChansLock.Lock()
		delete(server.CmdChans, command)
		server.CmdChansLock.Unlock()

		c <- result

		close(c)
	}
}

// EventNotify is an SSNTP callback stub for SsntpTestServer
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

	return
}
