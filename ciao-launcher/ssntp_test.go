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
	"sync"
	"testing"
	"time"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/01org/ciao/testutil"
)

type ssntpTestState struct {
	t       *testing.T
	status  bool
	error   ssntp.Error
	payload []byte
}

func (v *ssntpTestState) SendError(error ssntp.Error, payload []byte) (int, error) {
	v.error = error
	v.payload = payload
	return len(payload), nil
}

func (v *ssntpTestState) SendEvent(event ssntp.Event, payload []byte) (int, error) {
	return 0, nil
}

func (v *ssntpTestState) Dial(config *ssntp.Config, ntf ssntp.ClientNotifier) error {
	return nil
}

func (v *ssntpTestState) SendStatus(status ssntp.Status, payload []byte) (int, error) {
	return 0, nil
}

func (v *ssntpTestState) SendCommand(cmd ssntp.Command, payload []byte) (int, error) {
	return 0, nil
}

func (v *ssntpTestState) Role() ssntp.Role {
	return ssntp.AGENT | ssntp.NETAGENT
}

func (v *ssntpTestState) UUID() string {
	return testutil.AgentUUID
}

func (v *ssntpTestState) Close() {

}

func (v *ssntpTestState) isConnected() bool {
	return true
}

func (v *ssntpTestState) setStatus(status bool) {
	v.status = status
}

func (v *ssntpTestState) ClusterConfiguration() (payloads.Configure, error) {
	return payloads.Configure{}, nil
}

func TestAgentClientConnectDisconnectNotify(t *testing.T) {
	state := &ssntpTestState{}
	cmdCh := make(chan *cmdWrapper)
	ac := agentClient{conn: state, cmdCh: cmdCh}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		select {
		case cmd := <-cmdCh:
			if _, ok := cmd.cmd.(*statusCmd); !ok {
				t.Errorf("Unexpected command received.  Expected statusCmd")
			}
		case <-time.After(time.Second):
			t.Errorf("Timedout waiting for cmdCh")
		}
		wg.Done()
	}()

	ac.ConnectNotify()
	wg.Wait()
	if !state.status {
		t.Errorf("Connect Notify did not set agent client state")
	}

	ac.DisconnectNotify()
	if state.status {
		t.Errorf("Connect Notify did not set agent client state")
	}
}

func TestAgentClientStatusNotify(t *testing.T) {
	state := &ssntpTestState{}
	ac := agentClient{conn: state}
	ac.StatusNotify(ssntp.CONNECTED, nil)
}

func TestAgentClientEventNotify(t *testing.T) {
	state := &ssntpTestState{}
	ac := agentClient{conn: state}
	ac.EventNotify(ssntp.TenantAdded, nil)
}

func TestAgentClientErrorNotify(t *testing.T) {
	state := &ssntpTestState{}
	ac := agentClient{conn: state}
	ac.ErrorNotify(ssntp.InvalidFrameType, nil)
}

func checkErrorPayload(t *testing.T, ac *agentClient, state *ssntpTestState, cmd ssntp.Command,
	error ssntp.Error) {
	state.status = true
	frame := &ssntp.Frame{Payload: []byte{'h'}}
	ac.CommandNotify(cmd, frame)
	if state.error != error {
		t.Errorf("Expected SSNTP error %d", error)
	}
	if len(state.payload) == 0 {
		t.Errorf("Expected Non empty payload")
	}
}

func TestAgentClientStart(t *testing.T) {
	state := &ssntpTestState{}
	cmdCh := make(chan *cmdWrapper)
	ac := agentClient{conn: state, cmdCh: cmdCh}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		select {
		case cmd := <-cmdCh:
			if _, ok := cmd.cmd.(*insStartCmd); !ok {
				t.Errorf("Unexpected command received.  Expected startCmd")
			}
			if cmd.instance != testutil.InstanceUUID {
				t.Errorf("Unexpected instanced.  Expected %s found %s",
					testutil.InstanceUUID, cmd.instance)
			}
		case <-time.After(time.Second):
			t.Errorf("Timedout waiting for cmdCh")
		}
		wg.Done()
	}()

	frame := &ssntp.Frame{Payload: []byte(testutil.StartYaml)}
	ac.CommandNotify(ssntp.START, frame)
	wg.Wait()

	checkErrorPayload(t, &ac, state, ssntp.START, ssntp.StartFailure)
}

func TestAgentClientRestart(t *testing.T) {
	state := &ssntpTestState{}
	cmdCh := make(chan *cmdWrapper)
	ac := agentClient{conn: state, cmdCh: cmdCh}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		select {
		case cmd := <-cmdCh:
			if _, ok := cmd.cmd.(*insRestartCmd); !ok {
				t.Errorf("Unexpected command received.  Expected restartCmd")
			}
			if cmd.instance != testutil.InstanceUUID {
				t.Errorf("Unexpected instanced.  Expected %s found %s",
					testutil.InstanceUUID, cmd.instance)
			}
		case <-time.After(time.Second):
			t.Errorf("Timedout waiting for cmdCh")
		}
		wg.Done()
	}()

	frame := &ssntp.Frame{Payload: []byte(testutil.RestartYaml)}
	ac.CommandNotify(ssntp.RESTART, frame)
	wg.Wait()

	checkErrorPayload(t, &ac, state, ssntp.RESTART, ssntp.RestartFailure)
}

func TestAgentClientStop(t *testing.T) {
	state := &ssntpTestState{}
	cmdCh := make(chan *cmdWrapper)
	ac := agentClient{conn: state, cmdCh: cmdCh}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		select {
		case cmd := <-cmdCh:
			if _, ok := cmd.cmd.(*insStopCmd); !ok {
				t.Errorf("Unexpected command received.  Expected stopCmd")
			}
			if cmd.instance != testutil.InstanceUUID {
				t.Errorf("Unexpected instanced.  Expected %s found %s",
					testutil.InstanceUUID, cmd.instance)
			}
		case <-time.After(time.Second):
			t.Errorf("Timedout waiting for cmdCh")
		}
		wg.Done()
	}()

	frame := &ssntp.Frame{Payload: []byte(testutil.StopYaml)}
	ac.CommandNotify(ssntp.STOP, frame)
	wg.Wait()

	checkErrorPayload(t, &ac, state, ssntp.STOP, ssntp.StopFailure)
}

func TestAgentClientDelete(t *testing.T) {
	state := &ssntpTestState{}
	cmdCh := make(chan *cmdWrapper)
	ac := agentClient{conn: state, cmdCh: cmdCh}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		select {
		case cmd := <-cmdCh:
			if _, ok := cmd.cmd.(*insDeleteCmd); !ok {
				t.Errorf("Unexpected command received.  Expected deleteCmd")
			}
			if cmd.instance != testutil.InstanceUUID {
				t.Errorf("Unexpected instanced.  Expected %s found %s",
					testutil.InstanceUUID, cmd.instance)
			}
		case <-time.After(time.Second):
			t.Errorf("Timedout waiting for cmdCh")
		}
		wg.Done()
	}()

	frame := &ssntp.Frame{Payload: []byte(testutil.DeleteYaml)}
	ac.CommandNotify(ssntp.DELETE, frame)
	wg.Wait()

	checkErrorPayload(t, &ac, state, ssntp.DELETE, ssntp.DeleteFailure)
}

func TestAgentAttachVolume(t *testing.T) {
	state := &ssntpTestState{}
	cmdCh := make(chan *cmdWrapper)
	ac := agentClient{conn: state, cmdCh: cmdCh}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		select {
		case cmd := <-cmdCh:
			if _, ok := cmd.cmd.(*insAttachVolumeCmd); !ok {
				t.Errorf("Unexpected command received.  Expected attachVolumeCmd")
			}
			if cmd.instance != testutil.InstanceUUID {
				t.Errorf("Unexpected instanced.  Expected %s found %s",
					testutil.InstanceUUID, cmd.instance)
			}
		case <-time.After(time.Second):
			t.Errorf("Timedout waiting for cmdCh")
		}
		wg.Done()
	}()

	frame := &ssntp.Frame{Payload: []byte(testutil.AttachVolumeYaml)}
	ac.CommandNotify(ssntp.AttachVolume, frame)
	wg.Wait()

	checkErrorPayload(t, &ac, state, ssntp.AttachVolume, ssntp.AttachVolumeFailure)
}

func TestAgentDetachVolume(t *testing.T) {
	state := &ssntpTestState{}
	cmdCh := make(chan *cmdWrapper)
	ac := agentClient{conn: state, cmdCh: cmdCh}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		select {
		case cmd := <-cmdCh:
			if _, ok := cmd.cmd.(*insDetachVolumeCmd); !ok {
				t.Errorf("Unexpected command received.  Expected detachVolumeCmd")
			}
			if cmd.instance != testutil.InstanceUUID {
				t.Errorf("Unexpected instanced.  Expected %s found %s",
					testutil.InstanceUUID, cmd.instance)
			}
		case <-time.After(time.Second):
			t.Errorf("Timedout waiting for cmdCh")
		}
		wg.Done()
	}()

	frame := &ssntp.Frame{Payload: []byte(testutil.DetachVolumeYaml)}
	ac.CommandNotify(ssntp.DetachVolume, frame)
	wg.Wait()

	checkErrorPayload(t, &ac, state, ssntp.DetachVolume, ssntp.DetachVolumeFailure)
}
