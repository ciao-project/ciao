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
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"path"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/01org/ciao/osprepare"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/golang/glog"
)

var profileFN func() func()
var traceFN func() func()

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

var serverCertPath string
var clientCertPath string
var computeNet []string
var mgmtNet []string
var networking bool
var hardReset bool
var diskLimit bool
var memLimit bool
var cephID string
var simulate bool
var maxInstances = int(math.MaxInt32)

func init() {
	flag.StringVar(&serverCertPath, "cacert", "", "Client certificate")
	flag.StringVar(&clientCertPath, "cert", "", "CA certificate")
	flag.BoolVar(&networking, "network", true, "Enable networking")
	flag.BoolVar(&hardReset, "hard-reset", false, "Kill and delete all instances, reset networking and exit")
	flag.BoolVar(&simulate, "simulation", false, "Launcher simulation")
	flag.StringVar(&cephID, "ceph_id", "", "ceph client id")
}

const (
	lockDir        = "/tmp/lock/ciao"
	instancesDir   = "/var/lib/ciao/instances"
	logDir         = "/var/lib/ciao/logs/launcher"
	instanceState  = "state"
	lockFile       = "client-agent.lock"
	statsPeriod    = 6
	resourcePeriod = 30
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
	Role() ssntp.Role
	UUID() string
	Close()
	isConnected() bool
	setStatus(status bool)
	ClusterConfiguration() (payloads.Configure, error)
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
	case ssntp.AttachVolume:
		instance, volume, payloadErr := parseAttachVolumePayload(payload)
		if payloadErr != nil {
			attachVolumeError := &attachVolumeError{
				payloadErr.err,
				payloads.AttachVolumeFailureReason(payloadErr.code),
			}
			attachVolumeError.send(client.conn, "", "")
			glog.Errorf("Unable to parse YAML: %s", payloadErr.err)
			return
		}
		client.cmdCh <- &cmdWrapper{instance, &insAttachVolumeCmd{volume}}
	case ssntp.DetachVolume:
		instance, volume, payloadErr := parseDetachVolumePayload(payload)
		if payloadErr != nil {
			detachVolumeError := &detachVolumeError{
				payloadErr.err,
				payloads.DetachVolumeFailureReason(payloadErr.code),
			}
			detachVolumeError.send(client.conn, "", "")
			glog.Errorf("Unable to parse YAML: %s", payloadErr.err)
			return
		}
		client.cmdCh <- &cmdWrapper{instance, &insDetachVolumeCmd{volume}}
	}
}

func (client *agentClient) EventNotify(event ssntp.Event, frame *ssntp.Frame) {
	glog.Infof("EVENT %s", event)
}

func (client *agentClient) ErrorNotify(err ssntp.Error, frame *ssntp.Frame) {
	glog.Infof("ERROR %d", err)
}

func (client *agentClient) installLauncherDeps(doneCh chan struct{}) {
	ctx, cancelFunc := context.WithCancel(context.Background())

	ch := make(chan error)
	go func() {

		ospLogger := osprepare.OSPGlogLogger{}
		osprepare.Bootstrap(ctx, ospLogger)

		launcherDeps := osprepare.NewPackageRequirements()

		role := client.conn.Role()
		if role.IsNetAgent() {
			launcherDeps.Append(launcherNetNodeDeps)
		}
		if role.IsAgent() {
			launcherDeps.Append(launcherComputeNodeDeps)
		}

		osprepare.InstallDeps(ctx, launcherDeps, ospLogger)

		ch <- nil
	}()

	select {
	case <-doneCh:
		glog.Info("Received terminating signal.  Cancelling installation of launcher dependencies.")
		cancelFunc()
		<-ch
	case err := <-ch:
		if err != nil {
			glog.Errorf("Failed to install launcher dependencies: %v\n", err)
		}
		cancelFunc()
	}
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

func startNetwork(doneCh chan struct{}) error {
	if networking {
		ctx, cancelFunc := context.WithCancel(context.Background())
		ch := initNetworking(ctx)
		select {
		case <-doneCh:
			glog.Info("Received terminating signal.  Quitting")
			cancelFunc()
			return fmt.Errorf("Init network cancelled.")
		case err := <-ch:
			cancelFunc()
			if err != nil {
				glog.Errorf("Failed to init network: %v\n", err)
				return err
			}
		}
	}
	return nil
}

func loadClusterConfig(conn serverConn) error {
	clusterConfig, err := conn.ClusterConfiguration()
	if err != nil {
		return err
	}
	computeNet = clusterConfig.Configure.Launcher.ComputeNetwork
	mgmtNet = clusterConfig.Configure.Launcher.ManagementNetwork
	diskLimit = clusterConfig.Configure.Launcher.DiskLimit
	memLimit = clusterConfig.Configure.Launcher.MemoryLimit
	if cephID == "" {
		cephID = clusterConfig.Configure.Storage.CephID
	}
	return nil
}

func printClusterConfig() {
	glog.Info("Cluster Configuration")
	glog.Info("-----------------------")
	glog.Infof("Compute Network:      %v", computeNet)
	glog.Infof("Management Network:   %v", mgmtNet)
	glog.Infof("Disk Limit:           %v", diskLimit)
	glog.Infof("Memory Limit:         %v", memLimit)
	glog.Infof("Ceph ID:              %v", cephID)
}

func connectToServer(doneCh chan struct{}, statusCh chan struct{}) {

	defer func() {
		statusCh <- struct{}{}
	}()

	var wg sync.WaitGroup

	cfg := &ssntp.Config{CAcert: serverCertPath, Cert: clientCertPath,
		Log: ssntp.Log}
	client := &agentClient{
		conn:  &ssntpConn{},
		cmdCh: make(chan *cmdWrapper),
	}

	var ovsCh chan<- interface{}

	dialCh := make(chan error)

	go func() {
		err := client.conn.Dial(cfg, client)
		if err != nil {
			glog.Errorf("Unable to connect to server %v", err)
		}

		dialCh <- err
	}()

	select {
	case err := <-dialCh:
		if err != nil {
			break
		}

		role := client.conn.Role()
		if !(role.IsNetAgent() || role.IsAgent()) {
			glog.Errorf("Invalid certificate role: %s", role.String())
			client.conn.Close()
			break
		}

		err = loadClusterConfig(client.conn)
		if err != nil {
			glog.Errorf("Unable to get Cluster Configuration %v", err)
			client.conn.Close()
			break
		}
		printClusterConfig()

		client.installLauncherDeps(doneCh)

		err = startNetwork(doneCh)
		if err != nil {
			glog.Errorf("Failed to start network: %v\n", err)
			client.conn.Close()
			break
		}
		defer shutdownNetwork()

		ovsCh = startOverseer(&wg, client)
	case <-doneCh:
		client.conn.Close()
		<-dialCh
		return
	}

DONE:
	for {
		select {
		case <-doneCh:
			client.conn.Close()
			break DONE
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

	if ovsCh != nil {
		close(ovsCh)
	}
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
			debug.SetTraceback("all")
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

	glog.Info("Starting Launcher")

	exitCode := 0
	var stopProfile func()
	if profileFN != nil {
		stopProfile = profileFN()
	}

	var stopTrace func()
	if traceFN != nil {
		stopTrace = traceFN()
	}

	if hardReset {
		purgeLauncherState()
	} else {
		setLimits()

		glog.Infof("Launcher will allow a maximum of %d instances", maxInstances)

		if err := createMandatoryDirs(); err != nil {
			glog.Fatalf("Unable to create mandatory dirs: %v", err)
		}

		exitCode = startLauncher()
	}

	if stopTrace != nil {
		stopTrace()
	}

	if stopProfile != nil {
		stopProfile()
	}

	glog.Flush()
	glog.Info("Exit")

	os.Exit(exitCode)
}
