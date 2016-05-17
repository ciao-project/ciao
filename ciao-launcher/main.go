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
	"log"
	"math"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/context"

	"github.com/golang/glog"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
)

var profileFN func() func()

type networkFlag string

func (f *networkFlag) String() string {
	return string(*f)
}

func (f *networkFlag) Set(val string) error {
	if val != "none" && val != "cn" && val != "nn" && val != "dual" {
		return fmt.Errorf("none, cn, nn or dual expected")
	}
	*f = networkFlag(val)

	return nil
}

func (f *networkFlag) Enabled() bool {
	return string(*f) != "none"
}

func (f *networkFlag) NetworkNode() bool {
	return string(*f) == "nn"
}

func (f *networkFlag) DualMode() bool {
	return string(*f) == "dual"
}

type uiFlag string

func (f *uiFlag) String() string {
	return string(*f)
}

func (f *uiFlag) Set(val string) error {
	if val != "none" && val != "nc" && val != "spice" {
		return fmt.Errorf("none, nc or spice expected")
	}
	*f = uiFlag(val)

	return nil
}

func (f *uiFlag) Enabled() bool {
	return string(*f) != "none"
}

var serverURL string
var serverCertPath string
var clientCertPath string
var computeNet string
var mgmtNet string
var networking networkFlag = "none"
var hardReset bool
var diskLimit bool
var memLimit bool
var simulate bool
var maxInstances = int(math.MaxInt32)

func init() {
	flag.StringVar(&serverURL, "server", "", "URL of SSNTP server")
	flag.StringVar(&serverCertPath, "cacert", "/etc/pki/ciao/CAcert-server-localhost.pem", "Client certificate")
	flag.StringVar(&clientCertPath, "cert", "/etc/pki/ciao/cert-client-localhost.pem", "CA certificate")
	flag.StringVar(&computeNet, "compute-net", "", "Compute Subnet")
	flag.StringVar(&mgmtNet, "mgmt-net", "", "Management Subnet")
	flag.Var(&networking, "network", "Can be none, cn (compute node) or nn (network node)")
	flag.BoolVar(&hardReset, "hard-reset", false, "Kill and delete all instances, reset networking and exit")
	flag.BoolVar(&diskLimit, "disk-limit", true, "Use disk usage limits")
	flag.BoolVar(&memLimit, "mem-limit", true, "Use memory usage limits")
	flag.BoolVar(&simulate, "simulation", false, "Launcher simulation")
}

const (
	lockDir       = "/tmp/lock/ciao"
	instancesDir  = "/var/lib/ciao/instances"
	logDir        = "/var/lib/ciao/logs/launcher"
	instanceState = "state"
	lockFile      = "client-agent.lock"
	statsPeriod   = 30
)

type cmdWrapper struct {
	instance string
	cmd      interface{}
}
type statusCmd struct{}

type serverConn interface {
	SendError(error ssntp.Error, payload []byte) (int, error)
	SendEvent(event ssntp.Event, payload []byte) (int, error)
	Dial(config *ssntp.Config, ntf ssntp.ClientNotifier) error
	SendStatus(status ssntp.Status, payload []byte) (int, error)
	SendCommand(cmd ssntp.Command, payload []byte) (int, error)
	UUID() string
	Close()
	isConnected() bool
	setStatus(status bool)
}

type ssntpConn struct {
	sync.RWMutex
	ssntp.Client
	connected bool
}

func (s *ssntpConn) isConnected() bool {
	s.RLock()
	defer s.RUnlock()
	return s.connected
}

func (s *ssntpConn) setStatus(status bool) {
	s.Lock()
	s.connected = status
	s.Unlock()
}

type agentClient struct {
	conn  serverConn
	cmdCh chan *cmdWrapper
}

func (client *agentClient) DisconnectNotify() {
	client.conn.setStatus(false)
	glog.Warning("disconnected")
}

func (client *agentClient) ConnectNotify() {
	client.conn.setStatus(true)
	client.cmdCh <- &cmdWrapper{"", &statusCmd{}}
	glog.Info("connected")
}

func (client *agentClient) StatusNotify(status ssntp.Status, frame *ssntp.Frame) {
	glog.Infof("STATUS %s", status)
}

func (client *agentClient) CommandNotify(cmd ssntp.Command, frame *ssntp.Frame) {
	payload := frame.Payload

	switch cmd {
	case ssntp.START:
		start, cn, md := splitYaml(payload)
		cfg, payloadErr := parseStartPayload(start)
		if payloadErr != nil {
			startError := &startError{
				payloadErr.err,
				payloads.StartFailureReason(payloadErr.code),
			}
			startError.send(client.conn, "")
			glog.Errorf("Unable to parse YAML: %v", payloadErr.err)
			return
		}
		client.cmdCh <- &cmdWrapper{cfg.Instance, &insStartCmd{cn, md, frame, cfg, time.Now()}}
	case ssntp.RESTART:
		instance, payloadErr := parseRestartPayload(payload)
		if payloadErr != nil {
			restartError := &restartError{
				payloadErr.err,
				payloads.RestartFailureReason(payloadErr.code),
			}
			restartError.send(client.conn, "")
			glog.Errorf("Unable to parse YAML: %v", payloadErr.err)
			return
		}
		client.cmdCh <- &cmdWrapper{instance, &insRestartCmd{}}
	case ssntp.STOP:
		instance, payloadErr := parseStopPayload(payload)
		if payloadErr != nil {
			stopError := &stopError{
				payloadErr.err,
				payloads.StopFailureReason(payloadErr.code),
			}
			stopError.send(client.conn, "")
			glog.Errorf("Unable to parse YAML: %s", payloadErr)
			return
		}
		client.cmdCh <- &cmdWrapper{instance, &insStopCmd{}}
	case ssntp.DELETE:
		instance, payloadErr := parseDeletePayload(payload)
		if payloadErr != nil {
			deleteError := &deleteError{
				payloadErr.err,
				payloads.DeleteFailureReason(payloadErr.code),
			}
			deleteError.send(client.conn, "")
			glog.Errorf("Unable to parse YAML: %s", payloadErr.err)
			return
		}
		client.cmdCh <- &cmdWrapper{instance, &insDeleteCmd{}}
	}
}

func (client *agentClient) EventNotify(event ssntp.Event, frame *ssntp.Frame) {
	glog.Infof("EVENT %s", event)
}

func (client *agentClient) ErrorNotify(err ssntp.Error, frame *ssntp.Frame) {
	glog.Infof("ERROR %d", err)
}

func insCmdChannel(instance string, ovsCh chan<- interface{}) chan<- interface{} {
	targetCh := make(chan ovsGetResult)
	ovsCh <- &ovsGetCmd{instance, targetCh}
	target := <-targetCh
	return target.cmdCh
}

func insState(instance string, ovsCh chan<- interface{}) ovsGetResult {
	targetCh := make(chan ovsGetResult)
	ovsCh <- &ovsGetCmd{instance, targetCh}
	return <-targetCh
}

func processCommand(conn serverConn, cmd *cmdWrapper, ovsCh chan<- interface{}) {
	var target chan<- interface{}
	var delCmd *insDeleteCmd

	switch insCmd := cmd.cmd.(type) {
	case *statusCmd:
		ovsCh <- &ovsStatsStatusCmd{}
		return
	case *insStartCmd:
		targetCh := make(chan ovsAddResult)
		ovsCh <- &ovsAddCmd{cmd.instance, insCmd.cfg, targetCh}
		addResult := <-targetCh
		if !addResult.canAdd {
			glog.Errorf("Instance will make node full: Disk %d Mem %d CPUs %d",
				insCmd.cfg.Disk, insCmd.cfg.Mem, insCmd.cfg.Cpus)
			se := startError{nil, payloads.FullComputeNode}
			se.send(conn, cmd.instance)
			return
		}
		target = addResult.cmdCh
	case *insDeleteCmd:
		insState := insState(cmd.instance, ovsCh)
		target = insState.cmdCh
		if target == nil {
			glog.Errorf("Instance %s does not exist", cmd.instance)
			de := deleteError{nil, payloads.DeleteNoInstance}
			de.send(conn, cmd.instance)
			return
		}
		delCmd = insCmd
		delCmd.running = insState.running
	case *insStopCmd:
		target = insCmdChannel(cmd.instance, ovsCh)
		if target == nil {
			glog.Errorf("Instance %s does not exist", cmd.instance)
			se := stopError{nil, payloads.StopNoInstance}
			se.send(conn, cmd.instance)
			return
		}
	case *insRestartCmd:
		target = insCmdChannel(cmd.instance, ovsCh)
		if target == nil {
			glog.Errorf("Instance %s does not exist", cmd.instance)
			re := restartError{nil, payloads.RestartNoInstance}
			re.send(conn, cmd.instance)
			return
		}
	default:
		target = insCmdChannel(cmd.instance, ovsCh)
	}

	if target == nil {
		glog.Errorf("Instance %s does not exist", cmd.instance)
		return
	}

	target <- cmd.cmd

	if delCmd != nil {
		errCh := make(chan error)
		ovsCh <- &ovsRemoveCmd{
			cmd.instance,
			delCmd.suicide,
			errCh}
		<-errCh
	}
}

func connectToServer(doneCh chan struct{}, statusCh chan struct{}) {

	defer func() {
		statusCh <- struct{}{}
	}()

	var wg sync.WaitGroup

	var role ssntp.Role
	if networking.NetworkNode() {
		role = ssntp.NETAGENT
	} else if networking.DualMode() {
		role = ssntp.AGENT | ssntp.NETAGENT
	} else {
		role = ssntp.AGENT
	}

	glog.Infof("Agent Role: %s", role.String())

	cfg := &ssntp.Config{URI: serverURL, CAcert: serverCertPath, Cert: clientCertPath,
		Log: ssntp.Log}
	client := &agentClient{
		conn:  &ssntpConn{},
		cmdCh: make(chan *cmdWrapper),
	}

	ovsCh := startOverseer(&wg, client)

	dialCh := make(chan error)

	go func() {
		err := client.conn.Dial(cfg, client)
		if err != nil {
			glog.Errorf("Unable to connect to server %v", err)
			dialCh <- err
			return
		}

		dialCh <- err
	}()

	dialing := true

DONE:
	for {
		select {
		case err := <-dialCh:
			dialing = false
			if err != nil {
				break DONE
			}
		case <-doneCh:
			client.conn.Close()
			if !dialing {
				break DONE
			}
		case cmd := <-client.cmdCh:
			/*
				Double check we're not quitting here.  Otherwise a flood of commands
				from the server could block our exit for an arbitrary amount of time,
				i.e, doneCh and cmdCh could become available at the same time.
			*/
			select {
			case <-doneCh:
				client.conn.Close()
				break DONE
			default:
			}

			processCommand(client.conn, cmd, ovsCh)
		}
	}

	close(ovsCh)
	wg.Wait()
	glog.Info("Overseer has closed down")
}

func getLock() error {
	err := os.MkdirAll(lockDir, 0777)
	if err != nil {
		glog.Errorf("Unable to create lockdir %s", lockDir)
		return err
	}

	/* We're going to let the OS close and unlock this fd */
	lockPath := path.Join(lockDir, lockFile)
	fd, err := syscall.Open(lockPath, syscall.O_CREAT, syscall.S_IWUSR|syscall.S_IRUSR)
	if err != nil {
		glog.Errorf("Unable to open lock file %v", err)
		return err
	}

	syscall.CloseOnExec(fd)

	if syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		glog.Error("Launcher is already running.  Exitting.")
		return fmt.Errorf("Unable to lock file %s", lockPath)
	}

	return nil
}

/* Must be called after flag.Parse() */
func initLogger() error {
	logDirFlag := flag.Lookup("log_dir")
	if logDirFlag == nil {
		return fmt.Errorf("log_dir does not exist")
	}

	if logDirFlag.Value.String() == "" {
		if err := logDirFlag.Value.Set(logDir); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(logDirFlag.Value.String(), 0755); err != nil {
		return fmt.Errorf("Unable to create log directory (%s) %v", logDir, err)
	}

	return nil
}

func createMandatoryDirs() error {
	if err := os.MkdirAll(instancesDir, 0755); err != nil {
		return fmt.Errorf("Unable to create instances directory (%s) %v",
			instancesDir, err)
	}

	return nil
}

func purgeLauncherState() {

	glog.Info("======= HARD RESET ======")

	glog.Info("Shutting down running instances")

	toRemove := make([]string, 0, 1024)
	networking := false
	dockerNetworking := false

	glog.Info("Init networking")

	if err := initNetworkPhase1(); err != nil {
		glog.Warningf("Failed to init network: %v\n", err)
	} else {
		networking = true
		defer shutdownNetwork()
		if err := initDockerNetworking(context.Background()); err != nil {
			glog.Info("Unable to initialise docker networking")
		} else {
			dockerNetworking = true
		}
	}

	_ = filepath.Walk(instancesDir, func(path string, info os.FileInfo, err error) error {
		if path == instancesDir {
			return nil
		}

		if !info.IsDir() {
			return nil
		}

		cfg, err := loadVMConfig(path)
		if err != nil {
			glog.Warningf("Unable to load config for %s: %v", path, err)
		} else {
			if cfg.Container {
				dockerKillInstance(path)
			} else {
				qemuKillInstance(path)
			}
		}
		toRemove = append(toRemove, path)
		return nil
	})

	for _, p := range toRemove {
		err := os.RemoveAll(p)
		if err != nil {
			glog.Warningf("Unable to remove instance dir for %s: %v", p, err)
		}
	}

	if dockerNetworking {
		glog.Info("Reset docker networking")

		resetDockerNetworking()
	}

	if !networking {
		return
	}

	glog.Info("Reset networking")

	err := cnNet.ResetNetwork()
	if err != nil {
		glog.Warningf("Unable to reset network: %v", err)
	}
}

func setLimits() {
	var rlim syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim)
	if err != nil {
		glog.Warningf("Getrlimit failed %v", err)
		return
	}

	glog.Infof("Initial nofile limits: cur %d max %d", rlim.Cur, rlim.Max)

	if rlim.Cur < rlim.Max {
		oldCur := rlim.Cur
		rlim.Cur = rlim.Max
		err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim)
		if err != nil {
			glog.Warningf("Setrlimit failed %v", err)
			rlim.Cur = oldCur
		}
	}

	glog.Infof("Updated nofile limits: cur %d max %d", rlim.Cur, rlim.Max)

	maxInstances = int(rlim.Cur / 5)
}

func startLauncher() int {
	doneCh := make(chan struct{})
	statusCh := make(chan struct{})
	signalCh := make(chan os.Signal, 1)
	timeoutCh := make(chan struct{})
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	if networking.Enabled() {
		ctx, cancelFunc := context.WithCancel(context.Background())
		ch := initNetworking(ctx)
		select {
		case <-signalCh:
			glog.Info("Received terminating signal.  Quitting")
			cancelFunc()
			return 1
		case err := <-ch:
			if err != nil {
				glog.Errorf("Failed to init network: %v\n", err)
				return 1
			}
		}

		defer shutdownNetwork()
	}

	go connectToServer(doneCh, statusCh)

DONE:
	for {
		select {
		case <-signalCh:
			glog.Info("Received terminating signal.  Waiting for server loop to quit")
			close(doneCh)
			go func() {
				time.Sleep(time.Second)
				timeoutCh <- struct{}{}
			}()
		case <-statusCh:
			glog.Info("Server Loop quit cleanly")
			break DONE
		case <-timeoutCh:
			glog.Warning("Server Loop did not exit within 1 second quitting")
			glog.Flush()

			/* We panic here to see which naughty go routines are still running. */

			panic("Server Loop did not exit within 1 second quitting")
		}
	}

	return 0
}

func main() {

	flag.Parse()

	if simulate == false && getLock() != nil {
		os.Exit(1)
	}

	if err := initLogger(); err != nil {
		log.Fatalf("Unable to initialise logs: %v", err)
	}

	if profileFN != nil {
		stopProfile := profileFN()
		if stopProfile != nil {
			defer stopProfile()
		}
	}

	defer func() {
		glog.Flush()
		glog.Info("Exit")
	}()

	glog.Info("Starting Launcher")

	if hardReset {
		purgeLauncherState()
		os.Exit(0)
	}

	setLimits()

	glog.Infof("Launcher will allow a maximum of %d instances", maxInstances)

	if err := createMandatoryDirs(); err != nil {
		glog.Fatalf("Unable to create mandatory dirs: %v", err)
	}

	os.Exit(startLauncher())
}
