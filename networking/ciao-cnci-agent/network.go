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

package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/golang/glog"

	"github.com/01org/ciao/networking/libsnnet"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
)

var gCnci *libsnnet.Cnci
var gFw *libsnnet.Firewall

//TODO: Subscribe to netlink event to monitor physical interface changes
//TODO: Why does go not allow chan interface{}
func initNetwork(cancelCh <-chan os.Signal) error {

	cnci := &libsnnet.Cnci{}

	cnci.NetworkConfig = &libsnnet.NetworkConfig{
		Mode: libsnnet.GreTunnel,
	}

	if computeNet != "" {
		_, cnet, _ := net.ParseCIDR(computeNet)
		if cnet == nil {
			return fmt.Errorf("Unable to Parse CIDR :" + computeNet)
		}
		cnci.ComputeNet = []net.IPNet{*cnet}
	}

	if mgmtNet != "" {
		_, mnet, _ := net.ParseCIDR(mgmtNet)
		if mnet == nil {
			return fmt.Errorf("Unable to Parse CIDR :" + mgmtNet)
		}
		cnci.ManagementNet = []net.IPNet{*mnet}
	}

	var err error
	delays := []int64{1, 2, 5, 10, 20, 40, 60}
	for _, d := range delays {
		err = cnci.Init()
		if err == nil {
			break
		}
		glog.Infof("cnci network failed %v retrying in %v", err, d)
		select {
		case <-time.After(time.Duration(d) * time.Second):
		case <-cancelCh:
			glog.Infof("cnci.Init network cancelled %s", err.Error())
			return fmt.Errorf("cnci.Init cancelled")
		}
	}
	if err != nil {
		glog.Errorf("cnci.Init network failed %s", err.Error())
		return err
	}

	gCnci = cnci

	if enableNetwork {
		fw, err := libsnnet.InitFirewall(gCnci.ComputeLink[0].Attrs().Name)
		if err != nil {
			glog.Errorf("Firewall initialize failed %v", err)
		}
		gFw = fw
	}
	glog.Infof("Network Initialized %v", gCnci)

	return nil
}

func unmarshallSubnetParams(cmd *payloads.TenantAddedEvent) (*net.IPNet, int, net.IP, error) {
	const maxKey = ^uint32(0)

	_, snet, err := net.ParseCIDR(cmd.TenantSubnet)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("Invalid Remote subnet %s", err.Error())
	}

	cIP := net.ParseIP(cmd.AgentIP)
	if cIP == nil {
		return nil, 0, nil, fmt.Errorf("Invalid CN IP %s", cmd.ConcentratorIP)
	}

	//TODO
	//When we go away from a 1:1 subnet to key map remove this check
	//Today this ensures the sanity of the YAML and CN
	key := int(binary.LittleEndian.Uint32(snet.IP))
	subnetKey := cmd.SubnetKey
	if key != subnetKey {
		return nil, 0, nil, fmt.Errorf("Invalid Subnet Key %s %x", cmd.TenantSubnet, cmd.SubnetKey)
	}

	return snet, subnetKey, cIP, nil
}

func genIPsInSubnet(subnet net.IPNet) []net.IP {

	var allIPs []net.IP

	ip := subnet.IP.To4().Mask(subnet.Mask)

	//Calculate subnet size
	ones, bits := subnet.Mask.Size()
	if bits != 32 || ones > 30 || ones == 0 {
		return nil
	}
	subnetSize := ^(^0 << uint32(32-ones))
	subnetSize -= 3 //network, gateway and broadcast

	//Skip the network address and gateway
	ip[3] += 2
	startU32 := binary.BigEndian.Uint32(ip)

	//Generate all valid IPs in this subnet
	for i := 0; i < subnetSize; i++ {
		vIP := make(net.IP, net.IPv4len)
		binary.BigEndian.PutUint32(vIP, startU32+uint32(i))
		allIPs = append(allIPs, vIP)
	}
	return allIPs
}

func natSSHSubnet(action libsnnet.FwAction, subnet net.IPNet, intIf string, extIf string) error {

	err := gFw.ExtFwding(action, extIf, intIf)
	if err != nil {
		return fmt.Errorf("Error: NAT %v failed %v", action, err)
	}

	ips := genIPsInSubnet(subnet)
	for _, ip := range ips {
		extPort, err := libsnnet.DebugSSHPortForIP(ip)
		if err != nil {
			return fmt.Errorf("Error: ssh fwd %v failed %v", action, err)
		}
		glog.Infof("ssh fwd IP[%s] Port[%d] %d %d", ip, extPort, ip[2], ip[3])

		err = gFw.ExtPortAccess(action, "tcp", extIf, extPort, ip, 22)
		if err != nil {
			return fmt.Errorf("Error: ssh fwd %v failed %v", action, err)
		}
	}
	return nil
}

func addRemoteSubnet(cmd *payloads.TenantAddedEvent) error {
	rs, tk, rip, err := unmarshallSubnetParams(cmd)

	if err != nil {
		glog.Errorf("cnci.AddRemoteSubnet invalid params %s %x %s %s", rs, tk, rip, err)
		return err
	}

	if !enableNetwork {
		return nil
	}
	bridge, err := gCnci.AddRemoteSubnet(*rs, tk, rip)
	if err != nil {
		glog.Errorf("cnci.AddRemoteSubnet failed %s %x %s %s", rs, tk, rip, err)
		return err
	}

	glog.Infof("cnci.AddRemoteSubnet success %s %x %s", rs, tk, rip, err)

	if enableNATssh && bridge != "" {
		err = natSSHSubnet(libsnnet.FwEnable, *rs, bridge, gCnci.ComputeLink[0].Attrs().Name)
		if err != nil {
			glog.Errorf("enable ssh nat failed %s %x %s %s", rs, tk, bridge, err)
			return err
		}
		glog.Infof("cnci.AddRemoteSubnet ssh nat success %s %x %s", rs, tk, bridge, err)
	}
	return nil
}

func delRemoteSubnet(cmd *payloads.TenantAddedEvent) error {
	rs, tk, rip, err := unmarshallSubnetParams(cmd)

	if err != nil {
		glog.Errorf("cnci.delRemoteSubnet invalid params %s %x %s %s", rs, tk, rip, err)
		return err
	}

	if !enableNetwork {
		return nil
	}

	err = gCnci.DelRemoteSubnet(*rs, tk, rip)
	if err != nil {
		glog.Errorf("cnci.DelRemoteSubnet failed %s %x %s %s", rs, tk, rip, err)
		return err
	}
	glog.Infof("cnci.DelRemoteSubnet success %s %x %s", rs, tk, rip, err)

	/* We do not delete the bridge till reset.
	if enableNATssh {
		err = natSshSubnet(libsnnet.FwDisable, *rs, bridge, gCnci.ComputeLink[0].Attrs().Name)
		if err != nil {
			glog.Errorf("disable ssh nat failed %s %x %s %s", rs, tk, bridge, err)
			return err
		}
	}
	glog.Infof("cnci.DelRemoteSubnet ssh success %s %x %s", rs, tk, bridge, err)
	*/

	return nil
}

func cnciAddedMarshal(agentUUID string) ([]byte, error) {
	var cnciAdded payloads.EventConcentratorInstanceAdded
	evt := &cnciAdded.CNCIAdded

	//TODO: How do we set this up. evt.InstanceUUID = gCnci.ID
	evt.InstanceUUID = agentUUID
	evt.TenantUUID = gCnci.Tenant
	evt.ConcentratorIP = gCnci.ComputeAddr[0].IP.String()
	evt.ConcentratorMAC = gCnci.ComputeLink[0].Attrs().HardwareAddr.String()

	if evt.ConcentratorIP == "<nil>" || evt.ConcentratorMAC == "" {
		glog.Errorf("cnci.cnciAddedMarshal invalid physical configuration")
		return nil, fmt.Errorf("cnci.cnciAddedMarshal invalid physical configuration")
	}

	glog.Infoln("cnciAdded Event ", cnciAdded)

	return yaml.Marshal(&cnciAdded)
}

func publicIPAssignedMarshal(cmd *payloads.PublicIPCommand) ([]byte, error) {
	var publicIPAssigned payloads.EventPublicIPAssigned
	evt := &publicIPAssigned.AssignedIP

	evt.ConcentratorUUID = cmd.ConcentratorUUID
	evt.InstanceUUID = cmd.InstanceUUID
	evt.PublicIP = cmd.PublicIP
	evt.PrivateIP = cmd.PrivateIP

	glog.Infoln("PublicIPAssignedMarshal Event ", publicIPAssigned)

	return yaml.Marshal(&publicIPAssigned)
}

func publicIPUnassignedMarshal(cmd *payloads.PublicIPCommand) ([]byte, error) {
	var publicIPUnassigned payloads.EventPublicIPUnassigned
	evt := &publicIPUnassigned.UnassignedIP

	evt.ConcentratorUUID = cmd.ConcentratorUUID
	evt.InstanceUUID = cmd.InstanceUUID
	evt.PublicIP = cmd.PublicIP
	evt.PrivateIP = cmd.PrivateIP

	glog.Infoln("PublicIPUnassignedMarshal Event ", publicIPUnassigned)

	return yaml.Marshal(&publicIPUnassigned)
}

func sendNetworkEvent(client *ssntpConn, eventType ssntp.Event, eventInfo interface{}) error {

	if !client.isConnected() {
		return fmt.Errorf("Unable to send %s %v", eventType, eventInfo)
	}

	payload, err := generateNetEventPayload(eventType, eventInfo, client.UUID())
	if err != nil {
		return fmt.Errorf("Unable parse ssntpEvent %s %v", err, eventInfo)
	}

	n, err := client.SendEvent(eventType, payload)
	if err != nil {
		return fmt.Errorf("Unable to send %s %s %v %d", err.Error(), eventType, eventInfo, n)
	}

	return nil
}

func generateNetEventPayload(eventType ssntp.Event, eventInfo interface{}, agentUUID string) ([]byte, error) {

	switch eventType {
	case ssntp.ConcentratorInstanceAdded:
		glog.Infof("generating cnciAdded Event Payload %s", agentUUID)
		return cnciAddedMarshal(agentUUID)
	case ssntp.PublicIPAssigned:
		glog.Infof("generating publicIP Assigned Event Payload %v", eventInfo)
		cmd, ok := eventInfo.(*payloads.PublicIPCommand)
		if !ok {
			return nil, fmt.Errorf("PublicIPAssigned Invalid eventInfo [%T] %v", eventInfo, eventInfo)
		}
		return publicIPAssignedMarshal(cmd)
	case ssntp.PublicIPUnassigned:
		glog.Infof("generating publicIP Unassigned Event Payload %v", eventInfo)
		cmd, ok := eventInfo.(*payloads.PublicIPCommand)
		if !ok {
			return nil, fmt.Errorf("PublicIPUnassigned Invalid eventInfo [%T] %v", eventInfo, eventInfo)
		}
		return publicIPUnassignedMarshal(cmd)
	default:
		return nil, fmt.Errorf("Unsupported ssntpEventInfo type: %v", eventType)
	}

}

func unmarshallPubIP(cmd *payloads.PublicIPCommand) (net.IP, net.IP, error) {

	prIP := net.ParseIP(cmd.PrivateIP)
	puIP := net.ParseIP(cmd.PublicIP)

	switch {
	case prIP == nil:
		return nil, nil, fmt.Errorf("invalid private IP %v", cmd.PrivateIP)
	case puIP == nil:
		return nil, nil, fmt.Errorf("invalid public IP %v", cmd.PublicIP)
	}

	return prIP, puIP, nil

}

func assignPubIP(cmd *payloads.PublicIPCommand) error {

	prIP, puIP, err := unmarshallPubIP(cmd)
	if err != nil {
		return fmt.Errorf("cnci.assignPubIP invalid params %v %v", err, cmd)
	}

	err = gFw.PublicIPAccess(libsnnet.FwEnable, prIP, puIP, gCnci.ComputeLink[0].Attrs().Name)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	return nil
}

func releasePubIP(cmd *payloads.PublicIPCommand) error {

	prIP, puIP, err := unmarshallPubIP(cmd)
	if err != nil {
		return fmt.Errorf("cnci.releasePubIP invalid params %v %v", err, cmd)
	}

	err = gFw.PublicIPAccess(libsnnet.FwDisable, prIP, puIP, gCnci.ComputeLink[0].Attrs().Name)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	return nil
}
