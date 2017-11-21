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
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"context"

	"github.com/ciao-project/ciao/networking/libsnnet"
	"github.com/ciao-project/ciao/payloads"
	"github.com/ciao-project/ciao/ssntp"
	"github.com/golang/glog"
)

var cnNet *libsnnet.ComputeNode
var hostname string
var nicInfo []*payloads.NetworkStat
var dockerNet *libsnnet.DockerPlugin

type networkConfig struct {
	ComputeNet []string
	MgmtNet    []string
}

func (nc *networkConfig) Save() error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(nc); err != nil {
		return err
	}
	return ioutil.WriteFile(networkFile, buf.Bytes(), 0600)
}

func (nc *networkConfig) Load() error {
	data, err := ioutil.ReadFile(networkFile)
	if err != nil {
		return err
	}
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	return dec.Decode(nc)
}

func initNetworkPhase1() error {

	cn := &libsnnet.ComputeNode{}

	cnetList := make([]net.IPNet, len(netConfig.ComputeNet))
	for i, netStr := range netConfig.ComputeNet {
		_, cnet, _ := net.ParseCIDR(netStr)
		if cnet == nil {
			return fmt.Errorf("Unable to Parse CIDR :" + netStr)
		}
		cnetList[i] = *cnet
	}

	mnetList := make([]net.IPNet, len(netConfig.MgmtNet))
	for i, netStr := range netConfig.MgmtNet {
		_, mnet, _ := net.ParseCIDR(netStr)
		if mnet == nil {
			return fmt.Errorf("Unable to Parse CIDR :" + netStr)
		}
		mnetList[i] = *mnet
	}

	cn.NetworkConfig = &libsnnet.NetworkConfig{
		ManagementNet: mnetList,
		ComputeNet:    cnetList,
		Mode:          libsnnet.GreTunnel,
	}

	libsnnet.CnMaxAPIConcurrency = 1
	if err := cn.Init(); err != nil {
		return err
	}

	cnNet = cn

	return nil
}

func initDockerNetworking(ctx context.Context) error {
	dockerPlugin := libsnnet.NewDockerPlugin()
	if err := dockerPlugin.Init(); err != nil {
		glog.Warningf("Docker Init failed: %v", err)
		return err
	}

	if err := dockerPlugin.Start(); err != nil {
		if err := dockerPlugin.Close(); err != nil {
			glog.Warningf("Failed to close docker plugin: %v ", err)
		}
		glog.Warningf("Docker start failed: %v ", err)
		return err
	}

	dockerNet = dockerPlugin

	return nil
}

func shutdownDockerNetwork() {
	if dockerNet == nil {
		return
	}

	if err := dockerNet.Stop(); err != nil {
		glog.Warningf("Docker stop failed: %v", err)
	}

	if err := dockerNet.Close(); err != nil {
		glog.Warningf("Docker close failed: %v", err)
	}

	glog.Infof("Docker networking shutdown successfully")
}

func shutdownNetwork() {
	shutdownDockerNetwork()
}

func initNetwork(ctx context.Context) error {

	if err := initNetworkPhase1(); err != nil {
		return err
	}

	if err := initDockerNetworking(ctx); err != nil {
		glog.Warning("Unable to initialise docker networking")
	}

	if err := cnNet.DbRebuild(nil); err != nil {
		return err
	}

	limit := len(cnNet.ComputeAddr)
	if len(cnNet.ComputeLink) < limit {
		limit = len(cnNet.ComputeLink)
	}

	for i := 0; i < limit; i++ {
		nicInfo = append(nicInfo, &payloads.NetworkStat{
			NodeIP:  cnNet.ComputeAddr[i].IP.String(),
			NodeMAC: cnNet.ComputeLink[i].Attrs().HardwareAddr.String(),
		})
		glog.Infof("Network card %d Info", i)
		glog.Infof("  IP address of node is %s", nicInfo[i].NodeIP)
		glog.Infof("  MAC address of node is %s", nicInfo[i].NodeMAC)
	}

	if len(nicInfo) == 0 {
		glog.Warning("Unable to determine IP address. Should not happen")
	}

	var err error
	hostname, err = os.Hostname()
	if err == nil {
		glog.Infof("Hostname of node is %s", hostname)
	} else {
		glog.Warning("Unable to determine hostname %s", err)
	}

	return nil
}

func initNetworking(ctx context.Context) chan error {
	ch := make(chan error)
	go func() {
		err := initNetwork(ctx)
		ch <- err
	}()
	return ch
}

func createCNVnicCfg(cfg *vmConfig) (*libsnnet.VnicConfig, error) {

	glog.Info("Creating CN Vnic CFG")

	mac, err := net.ParseMAC(cfg.VnicMAC)
	if err != nil {
		return nil, fmt.Errorf("Invalid mac address %v", err)
	}

	_, vnet, err := net.ParseCIDR(cfg.SubnetIP)
	if err != nil {
		return nil, fmt.Errorf("Invalid vnic subnet %v", err)
	}

	concIP := net.ParseIP(cfg.ConcIP)
	if concIP == nil {
		return nil, fmt.Errorf("Invalid concentrator ip %s", cfg.ConcIP)
	}

	vnicIP := net.ParseIP(cfg.VnicIP)
	if vnicIP == nil {
		return nil, fmt.Errorf("Invalid vnicIP ip %s", cfg.VnicIP)
	}

	subnetKey := binary.LittleEndian.Uint32(vnet.IP)
	var role libsnnet.VnicRole
	if cfg.Container {
		role = libsnnet.TenantContainer
	} else {
		role = libsnnet.TenantVM
	}

	return &libsnnet.VnicConfig{
		VnicRole:   role,
		VnicIP:     vnicIP,
		ConcIP:     concIP,
		VnicMAC:    mac,
		Subnet:     *vnet,
		SubnetKey:  int(subnetKey),
		VnicID:     cfg.VnicUUID,
		InstanceID: cfg.Instance,
		TenantID:   cfg.TenantUUID,
		SubnetID:   cfg.SubnetIP,
		ConcID:     cfg.ConcUUID,
		Queues:     1,
	}, nil
}

func createCNCIVnicCfg(cfg *vmConfig) (*libsnnet.VnicConfig, error) {

	glog.Info("Creating CNCI Vnic CFG")

	mac, err := net.ParseMAC(cfg.VnicMAC)
	if err != nil {
		return nil, fmt.Errorf("Invalid mac address %v", err)
	}

	return &libsnnet.VnicConfig{
		VnicRole:   libsnnet.DataCenter,
		VnicMAC:    mac,
		VnicID:     cfg.VnicUUID,
		InstanceID: cfg.Instance,
		TenantID:   cfg.TenantUUID}, nil
}

func createVnicCfg(cfg *vmConfig) (*libsnnet.VnicConfig, error) {
	if cfg.NetworkNode {
		return createCNCIVnicCfg(cfg)
	}

	return createCNVnicCfg(cfg)
}

func sendNetworkEvent(conn serverConn, eventType ssntp.Event,
	event *libsnnet.SsntpEventInfo) {

	if event == nil || !conn.isConnected() {
		return
	}

	payload, err := generateNetEventPayload(event, conn.UUID())
	if err != nil {
		glog.Warningf("Unable parse ssntpEvent %s", err)
		return
	}

	_, err = conn.SendEvent(eventType, payload)
	if err != nil {
		glog.Warningf("Unable to send %s", event)
	}
}

func createVnic(conn serverConn, vnicCfg *libsnnet.VnicConfig) (string, string, string, []*os.File, error) {
	var name string
	var bridge string
	var gatewayIP string
	var fds []*os.File

	//BUG(markus): This function needs a context parameter

	if vnicCfg.VnicRole != libsnnet.DataCenter {
		var vnic *libsnnet.Vnic
		var event *libsnnet.SsntpEventInfo
		var info *libsnnet.ContainerInfo
		var err error
		if vnicCfg.VnicRole == libsnnet.TenantContainer {
			vnic, event, info, err = createDockerVnic(vnicCfg)
			if err != nil {
				glog.Errorf("cn.CreateVnic failed %v", err)
				return "", "", "", nil, err
			}
			bridge = info.SubnetID
		} else {
			vnic, event, info, err = cnNet.CreateVnic(vnicCfg)
			if err != nil {
				glog.Errorf("cn.CreateVnic failed %v", err)
				return "", "", "", nil, err
			}
			fds = vnic.FDs
		}
		sendNetworkEvent(conn, ssntp.TenantAdded, event)
		name = vnic.LinkName
		gatewayIP = info.Gateway.String()
		glog.Infoln("CN VNIC created =", name, info, event)
	} else {
		vnic, err := cnNet.CreateCnciVnic(vnicCfg)
		if err != nil {
			glog.Errorf("cn.CreateCnciVnic failed %v", err)
			return "", "", "", nil, err
		}
		name = vnic.LinkName
		glog.Infoln("CNCI VNIC created =", name)
	}

	return name, bridge, gatewayIP, fds, nil
}

func destroyVnic(conn serverConn, vnicCfg *libsnnet.VnicConfig) error {
	if vnicCfg.VnicRole != libsnnet.DataCenter {
		event, info, err := cnNet.DestroyVnic(vnicCfg)
		if err != nil {
			glog.Errorf("cn.DestroyVnic failed %v", err)
			return err
		}

		if info != nil && info.CNContainerEvent == libsnnet.ContainerNetworkDel {
			// This is one of these weird cases we will have with
			// docker in which some launcher and libssnet state gets out of
			// sync with docker.  Launcher needs a cleanup routine that detects
			// these inconsistencies and cleans up:
			// https://github.com/ciao-project/ciao/issues/4
			_ = destroyDockerNetwork(context.Background(), info.SubnetID)
		}

		sendNetworkEvent(conn, ssntp.TenantRemoved, event)

		glog.Infoln("CN VNIC Destroyed =", vnicCfg.VnicIP, event)
	} else {
		err := cnNet.DestroyCnciVnic(vnicCfg)
		if err != nil {
			glog.Errorf("cn.DestroyCnciVnic failed %v", err)
			return err
		}

		glog.Infoln("CNCI VNIC Destroyed =", vnicCfg.VnicIP)
	}

	return nil
}

func getNodeIPAddress() string {
	if len(nicInfo) == 0 {
		return "127.0.0.1"
	}

	return nicInfo[0].NodeIP
}
