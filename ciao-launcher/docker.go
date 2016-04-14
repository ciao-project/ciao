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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/version"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/filters"
	"github.com/docker/engine-api/types/network"
	"github.com/golang/glog"
	"golang.org/x/net/context"
)

var dockerClient struct {
	sync.Mutex
	cli *client.Client
}

type docker struct {
	cfg            *vmConfig
	instanceDir    string
	dockerID       string
	prevCPUTime    int64
	prevSampleTime time.Time
	pid            int
}

// It's not entirely clear that it's safe to call a client.Client object from
// multiple go routines simulataneously.  The code looks like it is re-entrant
// but this doesn't seem to be documented anywhere.  Need to check this.

// There's no real way to return an error from init at the moment, so we'll
// try to retrieve the client object at each new invocation of the virtualizer.

// BUG(markus): We shouldn't report ssh ports for docker instances

func getDockerClient() (cli *client.Client, err error) {
	dockerClient.Lock()
	if dockerClient.cli == nil {
		defaultHeaders := map[string]string{"User-Agent": "ciao-1.0"}
		dockerClient.cli, err = client.NewClient("unix:///var/run/docker.sock",
			"v1.22", nil, defaultHeaders)
	}
	cli = dockerClient.cli
	dockerClient.Unlock()
	return cli, err
}

func (d *docker) init(cfg *vmConfig, instanceDir string) {
	d.cfg = cfg
	d.instanceDir = instanceDir
}

func (d *docker) checkBackingImage() error {
	glog.Infof("Checking backing docker image %s", d.cfg.Image)

	cli, err := getDockerClient()
	if err != nil {
		return err
	}

	args := filters.NewArgs()
	images, err := cli.ImageList(context.Background(),
		types.ImageListOptions{
			MatchName: d.cfg.Image,
			All:       false,
			Filters:   args,
		})

	if err != nil {
		glog.Infof("Called to ImageList for %s failed: %v", d.cfg.Image, err)
		return err
	}

	if len(images) == 0 {
		glog.Infof("Docker Image not found %s", d.cfg.Image)
		return errImageNotFound
	}

	glog.Infof("Docker Image %s is present on node", d.cfg.Image)

	return nil
}

func (d *docker) downloadBackingImage() error {
	glog.Infof("Downloading backing docker image %s", d.cfg.Image)

	cli, err := getDockerClient()
	if err != nil {
		return err
	}

	prog, err := cli.ImagePull(context.Background(), types.ImagePullOptions{ImageID: d.cfg.Image}, nil)
	if err != nil {
		glog.Errorf("Unable to download image %s: %v\n", d.cfg.Image, err)
		return err

	}
	defer func() { _ = prog.Close() }()

	dec := json.NewDecoder(prog)
	var msg jsonmessage.JSONMessage
	err = dec.Decode(&msg)
	for err == nil {
		if msg.Error != nil {
			err = msg.Error
			break
		}

		err = dec.Decode(&msg)
	}

	if err != nil && err != io.EOF {
		glog.Errorf("Unable to download image %v\n", err)
		return err
	}

	return nil
}

func (d *docker) createImage(bridge string, userData, metaData []byte) error {
	var hostname string
	var cmd []string

	cli, err := getDockerClient()
	if err != nil {
		return err
	}

	md := &struct {
		Hostname string `json:"hostname"`
	}{}
	err = json.Unmarshal(metaData, md)
	if err != nil {
		glog.Info("Start command does not contain hostname. Setting to instance UUID")
		hostname = d.cfg.Instance
	} else {
		glog.Infof("Found hostname %s", md.Hostname)
		hostname = md.Hostname
	}

	ud := &struct {
		Cmds [][]string `yaml:"runcmd"`
	}{}
	err = yaml.Unmarshal(userData, ud)
	if err != nil {
		glog.Info("Start command does not contain a run command")
	} else {
		if len(ud.Cmds) >= 1 {
			cmd = ud.Cmds[0]
			if len(ud.Cmds) > 1 {
				glog.Warningf("Only one command supported.  Found %d in userdata", len(ud.Cmds))
			}
		}
	}

	config := &container.Config{
		Hostname: hostname,
		Image:    d.cfg.Image,
		Cmd:      cmd,
	}

	hostConfig := &container.HostConfig{}
	networkConfig := &network.NetworkingConfig{}
	if bridge != "" {
		config.MacAddress = d.cfg.VnicMAC
		hostConfig.NetworkMode = container.NetworkMode(bridge)
		networkConfig.EndpointsConfig = map[string]*network.EndpointSettings{
			bridge: &network.EndpointSettings{
				IPAMConfig: &network.EndpointIPAMConfig{
					IPv4Address: d.cfg.VnicIP,
				},
			},
		}
	}

	resp, err := cli.ContainerCreate(context.Background(), config, hostConfig, networkConfig,
		d.cfg.Instance)
	if err != nil {
		glog.Errorf("Unable to create container %v", err)
		return err
	}

	idPath := path.Join(d.instanceDir, "docker-id")
	err = ioutil.WriteFile(idPath, []byte(resp.ID), 0600)
	if err != nil {
		glog.Errorf("Unable to store docker container ID %v", err)
		return err
	}

	d.dockerID = resp.ID

	// This value is configurable.  Need to figure out how to get it from docker.

	d.cfg.Disk = 10000

	return nil
}

func (d *docker) deleteImage() error {
	if d.dockerID == "" {
		return nil
	}

	cli, err := getDockerClient()
	if err != nil {
		return err
	}

	err = cli.ContainerRemove(context.Background(),
		types.ContainerRemoveOptions{
			ContainerID: d.dockerID,
			Force:       true})
	if err != nil {
		glog.Warningf("Unable to delete docker instance %s:%s err %v",
			d.cfg.Instance, d.dockerID, err)
	}

	return err
}

func (d *docker) startVM(vnicName, ipAddress string) error {
	cli, err := getDockerClient()
	if err != nil {
		return err
	}

	err = cli.ContainerStart(context.Background(), d.dockerID)
	if err != nil {
		glog.Errorf("Unable to start container %v", err)
		return err
	}
	return nil
}

func dockerConnect(dockerChannel chan string, instance, dockerID string, closedCh chan struct{},
	connectedCh chan struct{}, wg *sync.WaitGroup, boot bool) {

	defer func() {
		if closedCh != nil {
			close(closedCh)
		}
		glog.Infof("Monitor function for %s exitting", instance)
		wg.Done()
	}()

	cli, err := getDockerClient()
	if err != nil {
		return
	}

	// BUG(markus): Need a way to cancel this.  Can't do this until we have contexts

	con, err := cli.ContainerInspect(context.Background(), dockerID)
	if err != nil {
		glog.Errorf("Unable to determine status of instance %s:%s: %v", instance, dockerID, err)
		return
	}

	if !con.State.Running && !con.State.Paused && !con.State.Restarting {
		glog.Infof("Docker Instance %s:%s is not running", instance, dockerID)
		return
	}

	close(connectedCh)

	ctx, cancelFunc := context.WithCancel(context.Background())
	lostContainerCh := make(chan struct{})
	go func() {
		defer close(lostContainerCh)
		if err != nil {
			return
		}
		ret, err := cli.ContainerWait(ctx, dockerID)
		glog.Infof("Instance %s:%s exitted with code %d err %v",
			instance, dockerID, ret, err)
	}()

DONE:
	for {
		select {
		case _, _ = <-lostContainerCh:
			break DONE
		case cmd, ok := <-dockerChannel:
			if !ok {
				glog.Info("Cancelling Wait")
				cancelFunc()
				_ = <-lostContainerCh
				break DONE
			} else if cmd == virtualizerStopCmd {
				err := cli.ContainerKill(context.Background(), dockerID, "KILL")
				if err != nil {
					glog.Errorf("Unable to stop instance %s:%s", instance, dockerID)
				}
			}
		}
	}

	glog.Infof("Docker Instance %s:%s shut down", instance, dockerID)
}

func (d *docker) monitorVM(closedCh chan struct{}, connectedCh chan struct{},
	wg *sync.WaitGroup, boot bool) chan string {

	if d.dockerID == "" {
		idPath := path.Join(d.instanceDir, "docker-id")
		data, err := ioutil.ReadFile(idPath)
		if err != nil {
			// We'll return an error later on in dockerConnect
			glog.Errorf("Unable to read docker container ID %v", err)
		} else {
			d.dockerID = string(data)
			glog.Infof("Instance UUID %s -> Docker UUID %s", d.cfg.Instance, d.dockerID)
		}
	}
	dockerChannel := make(chan string)
	wg.Add(1)
	go dockerConnect(dockerChannel, d.cfg.Instance, d.dockerID, closedCh, connectedCh, wg, boot)
	return dockerChannel
}

func (d *docker) computeInstanceDiskspace() int {
	if d.dockerID == "" {
		return -1
	}

	cli, err := getDockerClient()
	if err != nil {
		return -1
	}

	con, _, err := cli.ContainerInspectWithRaw(context.Background(), d.dockerID, true)
	if err != nil {
		glog.Errorf("Unable to determine status of instance %s:%s: %v", d.cfg.Instance,
			d.dockerID, err)
		return -1
	}

	if con.SizeRootFs == nil {
		return -1
	}

	return int(*con.SizeRootFs / 1000000)
}

func (d *docker) stats() (disk, memory, cpu int) {
	disk = d.computeInstanceDiskspace()
	memory = -1
	cpu = -1

	if d.pid == 0 {
		return
	}

	memory = computeProcessMemUsage(d.pid)
	if d.cfg == nil {
		return
	}

	cpuTime := computeProcessCPUTime(d.pid)
	now := time.Now()
	if d.prevCPUTime != -1 {
		cpu = int((100 * (cpuTime - d.prevCPUTime) /
			now.Sub(d.prevSampleTime).Nanoseconds()))
		if d.cfg.Cpus > 1 {
			cpu /= d.cfg.Cpus
		}
		// if glog.V(1) {
		//     glog.Infof("cpu %d%%\n", cpu)
		// }
	}
	d.prevCPUTime = cpuTime
	d.prevSampleTime = now

	return
}

func (d *docker) connected() {
	d.prevCPUTime = -1
	if d.pid == 0 {
		cli, err := getDockerClient()
		if err != nil {
			return
		}

		con, err := cli.ContainerInspect(context.Background(), d.dockerID)
		if err != nil {
			glog.Errorf("Unable to determine status of instance %s:%s: %v", d.cfg.Instance,
				d.dockerID, err)
			return
		}
		if con.State.Pid <= 0 {
			return
		}
		d.pid = con.State.Pid
	}
}

func (d *docker) lostVM() {
	d.pid = 0
	d.prevCPUTime = -1
}

//BUG(markus): Everything from here onwards should be in a different file.  It's confusing

func dockerKillInstance(instanceDir string) {
	idPath := path.Join(instanceDir, "docker-id")
	data, err := ioutil.ReadFile(idPath)
	if err != nil {
		glog.Errorf("Unable to read docker container ID %v", err)
		return
	}

	cli, err := getDockerClient()
	if err != nil {
		return
	}

	dockerID := string(data)
	err = cli.ContainerRemove(context.Background(),
		types.ContainerRemoveOptions{
			ContainerID: dockerID,
			Force:       true})
	if err != nil {
		glog.Warningf("Unable to delete docker instance %s err %v", dockerID, err)
	}
}

func checkDockerServerVersion(requiredVersion string, ctx context.Context) error {

	cli, err := getDockerClient()
	if err != nil {
		return err
	}

	ver, err := cli.ServerVersion(ctx)
	if err != nil {
		glog.Errorf("Unable to retrieve info from docker server err: %v", err)
		return err
	}

	glog.Infof("Docker server version %s", ver.Version)

	if version.Version(ver.Version).LessThan(version.Version(requiredVersion)) {
		return fmt.Errorf("Docker is too old.  Required >= %s.  Found %s.  Some things might not work.",
			requiredVersion, ver.Version)
	}

	return nil
}
