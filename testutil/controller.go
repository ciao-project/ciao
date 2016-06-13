// // Copyright (c) 2016 Intel Corporation
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

// SsntpTestController is global state for the testutil SSNTP controller
type SsntpTestController struct {
	Ssntp          ssntp.Client
	Name           string
	UUID           string
	CmdChans       map[ssntp.Command]chan Result
	CmdChansLock   *sync.Mutex
	EventChans     map[ssntp.Event]chan Result
	EventChansLock *sync.Mutex
	ErrorChans     map[ssntp.Error]chan Result
	ErrorChansLock *sync.Mutex
}

// NewSsntpTestControllerConnection creates an SsntpTestController and dials the server.
// Calling with a unique name parameter string for inclusion in the
// SsntpTestClient.Name field aides in debugging.  The uuid string
// parameter allows tests to specify a known uuid for simpler tests.
func NewSsntpTestControllerConnection(name string, uuid string) (*SsntpTestController, error) {
	if uuid == "" {
		return nil, errors.New("no uuid specified")
	}

	var role ssntp.Role = ssntp.Controller
	ctl := &SsntpTestController{
		Name: "Test " + role.String() + " " + name,
		UUID: uuid,
	}

	ctl.CmdChans = make(map[ssntp.Command]chan Result)
	ctl.CmdChansLock = &sync.Mutex{}
	ctl.EventChans = make(map[ssntp.Event]chan Result)
	ctl.EventChansLock = &sync.Mutex{}
	ctl.ErrorChans = make(map[ssntp.Error]chan Result)
	ctl.ErrorChansLock = &sync.Mutex{}

	config := &ssntp.Config{
		URI:    "",
		CAcert: ssntp.DefaultCACert,
		Cert:   ssntp.RoleToDefaultCertName(ssntp.Controller),
		Log:    ssntp.Log,
		UUID:   ctl.UUID,
	}

	if err := ctl.Ssntp.Dial(config, ctl); err != nil {
		return nil, err
	}
	return ctl, nil
}

// AddCmdChan adds a command to the SsntpTestController command channel
func (ctl *SsntpTestController) AddCmdChan(cmd ssntp.Command) *chan Result {
	c := make(chan Result)

	ctl.CmdChansLock.Lock()
	ctl.CmdChans[cmd] = c
	ctl.CmdChansLock.Unlock()

	return &c
}

// GetCmdChanResult gets a CmdResult from the SsntpTestController command channel
func (ctl *SsntpTestController) GetCmdChanResult(c *chan Result, cmd ssntp.Command) (result Result, err error) {
	select {
	case result = <-*c:
		if result.Err != nil {
			err = fmt.Errorf("Controller error sending %s command: %s\n", cmd, result.Err)
		}
	case <-time.After(5 * time.Second):
		err = fmt.Errorf("Timeout waiting for controller %s command result\n", cmd)
	}

	return result, err
}

// SendResultAndDelCmdChan deletes a command from the SsntpTestController command channel
func (ctl *SsntpTestController) SendResultAndDelCmdChan(cmd ssntp.Command, result Result) {
	ctl.CmdChansLock.Lock()
	defer ctl.CmdChansLock.Unlock()
	c, ok := ctl.CmdChans[cmd]
	if ok {
		delete(ctl.CmdChans, cmd)
		c <- result
		close(c)
	}
}

// AddEventChan adds a command to the SsntpTestController event channel
func (ctl *SsntpTestController) AddEventChan(evt ssntp.Event) *chan Result {
	c := make(chan Result)

	ctl.EventChansLock.Lock()
	ctl.EventChans[evt] = c
	ctl.EventChansLock.Unlock()

	return &c
}

// GetEventChanResult gets a CmdResult from the SsntpTestController event channel
func (ctl *SsntpTestController) GetEventChanResult(c *chan Result, evt ssntp.Event) (result Result, err error) {
	select {
	case result = <-*c:
		if result.Err != nil {
			err = fmt.Errorf("Controller error sending %s event: %s\n", evt, result.Err)
		}
	case <-time.After(20 * time.Second):
		err = fmt.Errorf("Timeout waiting for controller %s event result\n", evt)
	}

	return result, err
}

// SendResultAndDelEventChan deletes an event from the SsntpTestController event channel
func (ctl *SsntpTestController) SendResultAndDelEventChan(evt ssntp.Event, result Result) {
	ctl.EventChansLock.Lock()
	defer ctl.EventChansLock.Unlock()
	c, ok := ctl.EventChans[evt]
	if ok {
		delete(ctl.EventChans, evt)
		c <- result
		close(c)
	}
}

// AddErrorChan adds a command to the SsntpTestController error channel
func (ctl *SsntpTestController) AddErrorChan(error ssntp.Error) *chan Result {
	c := make(chan Result)

	ctl.ErrorChansLock.Lock()
	ctl.ErrorChans[error] = c
	ctl.ErrorChansLock.Unlock()

	return &c
}

// GetErrorChanResult gets a CmdResult from the SsntpTestController error channel
func (ctl *SsntpTestController) GetErrorChanResult(c *chan Result, error ssntp.Error) (result Result, err error) {
	select {
	case result = <-*c:
		if result.Err != nil {
			err = fmt.Errorf("Controller error sending %s error: %s\n", error, result.Err)
		}
	case <-time.After(20 * time.Second):
		err = fmt.Errorf("Timeout waiting for controller %s error result\n", error)
	}

	return result, err
}

// SendResultAndDelErrorChan deletes an error from the SsntpTestController error channel
func (ctl *SsntpTestController) SendResultAndDelErrorChan(error ssntp.Error, result Result) {
	ctl.ErrorChansLock.Lock()
	defer ctl.ErrorChansLock.Unlock()
	c, ok := ctl.ErrorChans[error]
	if ok {
		delete(ctl.ErrorChans, error)
		c <- result
		close(c)
	}
}

// ConnectNotify implements the SSNTP client ConnectNotify callback for SsntpTestController
func (ctl *SsntpTestController) ConnectNotify() {
	var result Result

	ctl.EventChansLock.Lock()
	defer ctl.EventChansLock.Unlock()
	c, ok := ctl.EventChans[ssntp.NodeConnected]
	if ok {
		delete(ctl.EventChans, ssntp.NodeConnected)
		c <- result
		close(c)
	}
}

// DisconnectNotify implements the SSNTP client DisconnectNotify callback for SsntpTestController
func (ctl *SsntpTestController) DisconnectNotify() {
	var result Result

	ctl.EventChansLock.Lock()
	defer ctl.EventChansLock.Unlock()
	c, ok := ctl.EventChans[ssntp.NodeDisconnected]
	if ok {
		delete(ctl.EventChans, ssntp.NodeDisconnected)
		c <- result
		close(c)
	}
}

// StatusNotify implements the SSNTP client StatusNotify callback for SsntpTestController
func (ctl *SsntpTestController) StatusNotify(status ssntp.Status, frame *ssntp.Frame) {
}

// CommandNotify implements the SSNTP client CommandNotify callback for SsntpTestController
func (ctl *SsntpTestController) CommandNotify(command ssntp.Command, frame *ssntp.Frame) {
	var result Result

	switch command {
	case ssntp.STATS:
		var stats payloads.Stat

		stats.Init()

		err := yaml.Unmarshal(frame.Payload, &stats)
		if err != nil {
			result.Err = err
		}
	default:
		fmt.Printf("controller unhandled command: %s\n", command.String())
	}

	ctl.CmdChansLock.Lock()
	defer ctl.CmdChansLock.Unlock()
	c, ok := ctl.CmdChans[command]
	if ok {
		delete(ctl.CmdChans, command)
		c <- result
		close(c)
	}
}

// EventNotify implements the SSNTP client EventNotify callback for SsntpTestController
func (ctl *SsntpTestController) EventNotify(event ssntp.Event, frame *ssntp.Frame) {
	var result Result

	switch event {
	case ssntp.InstanceDeleted:
		var deleteEvent payloads.EventInstanceDeleted

		err := yaml.Unmarshal(frame.Payload, &deleteEvent)
		if err != nil {
			result.Err = err
		}
	case ssntp.TraceReport:
		var traceEvent payloads.Trace

		err := yaml.Unmarshal(frame.Payload, &traceEvent)
		if err != nil {
			result.Err = err
		}
	case ssntp.ConcentratorInstanceAdded:
		var concentratorAddedEvent payloads.EventConcentratorInstanceAdded

		err := yaml.Unmarshal(frame.Payload, &concentratorAddedEvent)
		if err != nil {
			result.Err = err
		}
	default:
		fmt.Printf("controller unhandled event: %s\n", event.String())
	}

	ctl.EventChansLock.Lock()
	defer ctl.EventChansLock.Unlock()
	c, ok := ctl.EventChans[event]
	if ok {
		delete(ctl.EventChans, event)
		c <- result
		close(c)
	}
}

// ErrorNotify implements the SSNTP client ErrorNotify callback for SsntpTestController
func (ctl *SsntpTestController) ErrorNotify(error ssntp.Error, frame *ssntp.Frame) {
}
