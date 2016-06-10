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

package testutil_test

import (
	"os"
	"testing"
	"time"

	"github.com/01org/ciao/ssntp"
	. "github.com/01org/ciao/testutil"
	"github.com/docker/distribution/uuid"
)

var server SsntpTestServer
var controller *SsntpTestController
var agent *SsntpTestClient
var netAgent *SsntpTestClient

func TestStart(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.START)
	serverCh := server.AddCmdChan(ssntp.START)

	_, err := agent.Ssntp.SendCommand(ssntp.START, []byte(StartYaml))
	if err != nil {
		t.Fatal(err)
	}

	_, err = agent.GetCmdChanResult(agentCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.GetCmdChanResult(serverCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSendStatus(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.STATS)
	serverCh := server.AddCmdChan(ssntp.STATS)
	controllerCh := controller.AddCmdChan(ssntp.STATS)

	go agent.SendStats()

	_, err := agent.GetCmdChanResult(agentCh, ssntp.STATS)
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.GetCmdChanResult(serverCh, ssntp.STATS)
	if err != nil {
		t.Fatal(err)
	}
	_, err = controller.GetCmdChanResult(controllerCh, ssntp.STATS)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStartTraced(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.START)
	serverCh := server.AddCmdChan(ssntp.START)

	traceConfig := &ssntp.TraceConfig{
		PathTrace: true,
		Start:     time.Now(),
		Label:     []byte("testutilTracedSTART"),
	}

	_, err := agent.Ssntp.SendTracedCommand(ssntp.START, []byte(StartYaml), traceConfig)
	if err != nil {
		t.Fatal(err)
	}

	_, err = agent.GetCmdChanResult(agentCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.GetCmdChanResult(serverCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSendTrace(t *testing.T) {
	agentCh := agent.AddEventChan(ssntp.TraceReport)
	serverCh := server.AddEventChan(ssntp.TraceReport)

	go agent.SendTrace()

	_, err := agent.GetEventChanResult(agentCh, ssntp.TraceReport)
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.GetEventChanResult(serverCh, ssntp.TraceReport)
	if err != nil {
		t.Fatal(err)
	}
}

/*
func TestStartFailure(t *testing.T) {
	// do a start, but with bool fail == true (leads to a helper broken
	// out of starter aboeve
}

func TestRestartFailure(t *testing.T) {
	// agent.Ssntp.SendCommand(ssntp.RESTART, yaml)
	// ...yaml needs to include the instance UUID and the agent UUID
	// ...to get the agent UUID I need a stats
}

func TestStopFailure(t *testing.T) {
	// stop instance uuid which is not actually running
}
*/

func TestStartCNCI(t *testing.T) {
	netAgentCh := netAgent.AddCmdChan(ssntp.START)
	serverCh := server.AddCmdChan(ssntp.START)

	_, err := netAgent.Ssntp.SendCommand(ssntp.START, []byte(CNCIStartYaml))
	if err != nil {
		t.Fatal(err)
	}

	_, err = netAgent.GetCmdChanResult(netAgentCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.GetCmdChanResult(serverCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStop(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.STOP)
	serverCh := server.AddCmdChan(ssntp.STOP)

	_, err := agent.Ssntp.SendCommand(ssntp.STOP, []byte(StopYaml))
	if err != nil {
		t.Fatal(err)
	}

	_, err = agent.GetCmdChanResult(agentCh, ssntp.STOP)
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.GetCmdChanResult(serverCh, ssntp.STOP)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRestart(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.RESTART)
	serverCh := server.AddCmdChan(ssntp.RESTART)

	_, err := agent.Ssntp.SendCommand(ssntp.RESTART, []byte(RestartYaml))
	if err != nil {
		t.Fatal(err)
	}

	_, err = agent.GetCmdChanResult(agentCh, ssntp.RESTART)
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.GetCmdChanResult(serverCh, ssntp.RESTART)
	if err != nil {
		t.Fatal(err)
	}
}

func doDelete() error {
	agentCh := agent.AddCmdChan(ssntp.DELETE)
	serverCh := server.AddCmdChan(ssntp.DELETE)

	_, err := agent.Ssntp.SendCommand(ssntp.DELETE, []byte(DeleteYaml))
	if err != nil {
		return err
	}

	_, err = agent.GetCmdChanResult(agentCh, ssntp.DELETE)
	if err != nil {
		return err
	}
	_, err = server.GetCmdChanResult(serverCh, ssntp.DELETE)
	if err != nil {
		return err
	}
	return nil
}

func propagateInstanceDeleted() error {
	agentCh := agent.AddEventChan(ssntp.InstanceDeleted)
	serverCh := server.AddEventChan(ssntp.InstanceDeleted)
	controllerCh := controller.AddEventChan(ssntp.InstanceDeleted)

	go agent.SendDeleteEvent(InstanceUUID)

	_, err := agent.GetEventChanResult(agentCh, ssntp.InstanceDeleted)
	if err != nil {
		return err
	}
	_, err = server.GetEventChanResult(serverCh, ssntp.InstanceDeleted)
	if err != nil {
		return err
	}
	_, err = controller.GetEventChanResult(controllerCh, ssntp.InstanceDeleted)
	if err != nil {
		return err
	}
	return nil
}

func TestDelete(t *testing.T) {
	err := doDelete()
	if err != nil {
		t.Fatal(err)
	}

	err = propagateInstanceDeleted()
	if err != nil {
		t.Fatal(err)
	}
}

func stopServer() error {
	controllerCh := controller.AddEventChan(ssntp.NodeDisconnected)
	netAgentCh := netAgent.AddEventChan(ssntp.NodeDisconnected)
	agentCh := agent.AddEventChan(ssntp.NodeDisconnected)

	server.Ssntp.Stop()

	_, err := controller.GetEventChanResult(controllerCh, ssntp.NodeDisconnected)
	if err != nil {
		return err
	}
	_, err = netAgent.GetEventChanResult(netAgentCh, ssntp.NodeDisconnected)
	if err != nil {
		return err
	}
	_, err = agent.GetEventChanResult(agentCh, ssntp.NodeDisconnected)
	if err != nil {
		return err
	}
	return nil
}

func restartServer() error {
	controllerCh := controller.AddEventChan(ssntp.NodeConnected)
	netAgentCh := netAgent.AddEventChan(ssntp.NodeConnected)
	agentCh := agent.AddEventChan(ssntp.NodeConnected)

	StartTestServer(&server)

	//MUST be after StartTestServer becase the channels are initialized on start
	serverCh := server.AddEventChan(ssntp.NodeConnected)

	_, err := controller.GetEventChanResult(controllerCh, ssntp.NodeConnected)
	if err != nil {
		return err
	}
	_, err = netAgent.GetEventChanResult(netAgentCh, ssntp.NodeConnected)
	if err != nil {
		return err
	}
	_, err = agent.GetEventChanResult(agentCh, ssntp.NodeConnected)
	if err != nil {
		return err
	}
	_, err = server.GetEventChanResult(serverCh, ssntp.NodeConnected)
	if err != nil {
		return err
	}
	return nil
}

func TestReconnects(t *testing.T) {
	err := stopServer()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)

	err = restartServer()
	if err != nil {
		t.Fatal(err)
	}
}

func TestMain(m *testing.M) {
	var err error

	// start server
	StartTestServer(&server)

	// start controller
	controllerUUID := uuid.Generate().String()
	controller, err = NewSsntpTestControllerConnection("Controller Client", controllerUUID)
	if err != nil {
		os.Exit(1)
	}

	// start agent
	agent, err = NewSsntpTestClientConnection("AGENT Client", ssntp.AGENT, AgentUUID)
	if err != nil {
		os.Exit(1)
	}

	// start netagent
	netAgentUUID := uuid.Generate().String()
	netAgent, err = NewSsntpTestClientConnection("NETAGENT Client", ssntp.NETAGENT, netAgentUUID)
	if err != nil {
		os.Exit(1)
	}

	status := m.Run()

	// stop everybody
	time.Sleep(1 * time.Second)
	controller.Ssntp.Close()

	time.Sleep(1 * time.Second)
	netAgent.Ssntp.Close()

	time.Sleep(1 * time.Second)
	agent.Ssntp.Close()

	time.Sleep(1 * time.Second)
	server.Ssntp.Stop()

	os.Exit(status)
}
