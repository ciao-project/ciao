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
	"path"
	"sync"
	"time"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"

	"github.com/golang/glog"
)

type insStartCmd struct {
	userData []byte
	metaData []byte
	frame    *ssntp.Frame
	cfg      *vmConfig
}
type insRestartCmd struct{}
type insDeleteCmd struct {
	suicide bool
	running ovsRunningState
}
type insStopCmd struct{}
type insMonitorCmd struct{}

/*
This functions asks the server loop to kill the instance.  An instance
needs to request that the server loop kill it if Start fails completly.
As the serverLoop does not wait for the start command to complete, we wouldn't
want to do this, as it would mean all start commands execute in serial,
the serverLoop cannot detect this situation.  Thus the instance loop needs
to request it's own death.

The server loop is the only go routine that can kill the instance.  If the
instance kills itself, the serverLoop would lockup if a command came in for
that instance while it was shutting down.  The instance go routine cannot
send a command to the serverLoop directly as this could lead to deadlock.
So we must spawn a separate go routine to do this.  We also need to handle
the case that this go routine blocks for ever if the serverLoop is quit
by CTRL-C.  That's why we select on doneCh as well.  In this case,
the command will never be written to the serverLoop, our go routine will
exit, the instance will exit and then finally the overseer will quit.

There's always the possibility new commands will be received for the
instance while it is waiting to be killed.  We'll just fail those.
*/

func killMe(instance string, doneCh chan struct{}, ac *agentClient, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		cmd := &cmdWrapper{instance, &insDeleteCmd{suicide: true}}
		select {
		case ac.cmdCh <- cmd:
		case <-doneCh:
		}
		wg.Done()
	}()
}

func instanceLoop(cmdCh chan interface{}, instance string, cfg *vmConfig, wg *sync.WaitGroup, doneCh chan struct{}, ac *agentClient, ovsCh chan<- interface{}) {
	var instanceWg sync.WaitGroup
	var monitorCh chan string
	var connectedCh chan struct{}
	var monitorCloseCh chan struct{}
	var statsTimer <-chan time.Time
	var vm virtualizer

	if simulate == true {
		vm = &simulation{}
	} else if cfg.Container {
		vm = &docker{}
	} else {
		vm = &qemu{}
	}
	instanceDir := path.Join(instancesDir, instance)
	vm.init(cfg, instanceDir)
	shuttingDown := false

	d, m, c := vm.stats()
	ovsCh <- &ovsStatsUpdateCmd{instance, m, d, c}

DONE:
	for {
		select {
		case <-doneCh:
			break DONE
		case <-statsTimer:
			d, m, c := vm.stats()
			ovsCh <- &ovsStatsUpdateCmd{instance, m, d, c}
			statsTimer = time.After(time.Second * statsPeriod)
		case cmd := <-cmdCh:
			select {
			case <-doneCh:
				break DONE
			default:
			}

			switch cmd := cmd.(type) {
			case *insStartCmd:
				glog.Info("Found start command")
				if monitorCh != nil {
					startErr := &startError{nil, payloads.AlreadyRunning}
					glog.Errorf("Unable to start instance[%s]", string(startErr.code))
					startErr.send(&ac.ssntpConn, instance)
					continue
				}
				startErr := processStart(cmd, instanceDir, vm, &ac.ssntpConn)
				if startErr != nil {
					glog.Errorf("Unable to start instance[%s]: %v", string(startErr.code), startErr.err)
					startErr.send(&ac.ssntpConn, instance)

					if startErr.code == payloads.LaunchFailure {
						ovsCh <- &ovsStateChange{instance, ovsStopped}
					} else if startErr.code != payloads.InstanceExists {
						glog.Warningf("Unable to create VM instance: %s.  Killing it", instance)
						killMe(instance, doneCh, ac, &instanceWg)
						shuttingDown = true
					}
					continue
				}

				connectedCh = make(chan struct{})
				monitorCloseCh = make(chan struct{})
				monitorCh = vm.monitorVM(monitorCloseCh, connectedCh, &instanceWg, false)
				ovsCh <- &ovsStatusCmd{}
				if cmd.frame != nil && cmd.frame.PathTrace() {
					ovsCh <- &ovsTraceFrame{cmd.frame}
				}
			case *insRestartCmd:
				glog.Info("Found restart command")

				if shuttingDown {
					restartErr := &restartError{nil, payloads.RestartNoInstance}
					glog.Errorf("Unable to restart instance[%s]", string(restartErr.code))
					restartErr.send(&ac.ssntpConn, instance)
					continue
				}

				if monitorCh != nil {
					restartErr := &restartError{nil, payloads.RestartAlreadyRunning}
					glog.Errorf("Unable to restart instance[%s]", string(restartErr.code))
					restartErr.send(&ac.ssntpConn, instance)
					continue
				}

				restartErr := processRestart(instanceDir, vm, &ac.ssntpConn, cfg)

				if restartErr != nil {
					glog.Errorf("Unable to restart instance[%s]: %v", string(restartErr.code),
						restartErr.err)
					restartErr.send(&ac.ssntpConn, instance)
					continue
				}

				connectedCh = make(chan struct{})
				monitorCloseCh = make(chan struct{})
				monitorCh = vm.monitorVM(monitorCloseCh, connectedCh, &instanceWg, false)
			case *insMonitorCmd:
				connectedCh = make(chan struct{})
				monitorCloseCh = make(chan struct{})
				monitorCh = vm.monitorVM(monitorCloseCh, connectedCh, &instanceWg, true)
			case *insStopCmd:

				if shuttingDown {
					stopErr := &stopError{nil, payloads.StopNoInstance}
					glog.Errorf("Unable to stop instance[%s]", string(stopErr.code))
					stopErr.send(&ac.ssntpConn, instance)
					continue
				}

				if monitorCh == nil {
					stopErr := &stopError{nil, payloads.StopAlreadyStopped}
					glog.Errorf("Unable to stop instance[%s]", string(stopErr.code))
					stopErr.send(&ac.ssntpConn, instance)
					continue
				}
				glog.Infof("Powerdown %s", instance)
				monitorCh <- virtualizerStopCmd
			case *insDeleteCmd:

				if shuttingDown && !cmd.suicide {
					deleteErr := &deleteError{nil, payloads.DeleteNoInstance}
					glog.Errorf("Unable to delete instance[%s]", string(deleteErr.code))
					deleteErr.send(&ac.ssntpConn, instance)
					continue
				}

				if monitorCh != nil {
					glog.Infof("Powerdown %s before deleting", instance)
					monitorCh <- virtualizerStopCmd
					vm.lostVM()
				}

				_ = processDelete(vm, instanceDir, &ac.ssntpConn, cmd.running)

				if !cmd.suicide {
					ovsCh <- &ovsStatusCmd{}
				}

				break DONE
			default:
				glog.Warning("Unknown command")
			}
		case <-monitorCloseCh:
			// Means we've lost VM for now
			vm.lostVM()
			d, m, c := vm.stats()
			ovsCh <- &ovsStatsUpdateCmd{instance, m, d, c}

			glog.Infof("Lost VM instance: %s", instance)
			monitorCloseCh = nil
			connectedCh = nil
			close(monitorCh)
			monitorCh = nil
			statsTimer = nil
			ovsCh <- &ovsStateChange{instance, ovsStopped}
		case <-connectedCh:
			connectedCh = nil
			vm.connected()
			ovsCh <- &ovsStateChange{instance, ovsRunning}
			d, m, c := vm.stats()
			ovsCh <- &ovsStatsUpdateCmd{instance, m, d, c}
			statsTimer = time.After(time.Second * statsPeriod)
		}
	}

	if monitorCh != nil {
		close(monitorCh)
	}

	glog.Infof("Instance goroutine %s waiting for monitor to exit", instance)
	instanceWg.Wait()
	glog.Infof("Instance goroutine %s exitted", instance)
	wg.Done()
}

func startInstance(instance string, cfg *vmConfig, wg *sync.WaitGroup, doneCh chan struct{},
	ac *agentClient, ovsCh chan<- interface{}) chan<- interface{} {
	cmdCh := make(chan interface{})
	wg.Add(1)
	go instanceLoop(cmdCh, instance, cfg, wg, doneCh, ac, ovsCh)
	return cmdCh
}
