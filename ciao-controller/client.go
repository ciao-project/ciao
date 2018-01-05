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
	"encoding/json"
	"fmt"
	"time"

	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/payloads"
	"github.com/ciao-project/ciao/ssntp"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type controllerClient interface {
	ssntp.ClientNotifier
	StartTracedWorkload(config string, startTime time.Time, label string) error
	StartWorkload(config string) error
	DeleteInstance(instanceID string, nodeID string) error
	StopInstance(instanceID string, nodeID string) error
	RestartInstance(i *types.Instance, w *types.Workload, t *types.Tenant) error
	RemoveInstance(instanceID string)
	EvacuateNode(nodeID string) error
	RestoreNode(nodeID string) error
	Disconnect()
	mapExternalIP(t types.Tenant, m types.MappedIP) error
	unMapExternalIP(t types.Tenant, m types.MappedIP) error
	attachVolume(volID string, instanceID string, nodeID string) error
	ssntpClient() *ssntp.Client
	CNCIRefresh(cnciID string, cnciList []payloads.CNCINet) error
}

type ssntpClient struct {
	ctl   *controller
	ssntp ssntp.Client
	name  string
}

func (client *ssntpClient) ConnectNotify() {
	glog.Info(client.name, " connected")
}

func (client *ssntpClient) DisconnectNotify() {
	glog.Info(client.name, " disconnected")
}

func (client *ssntpClient) StatusNotify(status ssntp.Status, frame *ssntp.Frame) {
	glog.Info("STATUS for ", client.name)
}

func (client *ssntpClient) CommandNotify(command ssntp.Command, frame *ssntp.Frame) {
	var stats payloads.Stat
	payload := frame.Payload

	glog.Info("COMMAND ", command, " for ", client.name)

	if command == ssntp.STATS {
		stats.Init()
		err := yaml.Unmarshal(payload, &stats)
		if err != nil {
			glog.Warningf("Error unmarshalling STATS: %v", err)
			return
		}
		err = client.ctl.ds.HandleStats(stats)
		if err != nil {
			glog.Warningf("Error updating stats in datastore: %v", err)
		}
	}
	glog.V(1).Info(string(payload))
}

func (client *ssntpClient) deleteEphemeralStorage(instanceID string) {
	err := client.ctl.deleteEphemeralStorage(instanceID)
	if err != nil {
		glog.Warningf("Error deleting ephemeral storage for instance: %s: %v", instanceID, err)
	}
}

func (client *ssntpClient) releaseResources(instanceID string) error {
	i, err := client.ctl.ds.GetInstance(instanceID)
	if err != nil {
		return errors.Wrapf(err, "error getting instance from datastore")
	}

	// CNCI resources are not quota tracked
	if i.CNCI {
		return nil
	}

	wl, err := client.ctl.ds.GetWorkload(i.WorkloadID)
	if err != nil {
		return errors.Wrapf(err, "error getting workload for instance from datastore")
	}

	resources := []payloads.RequestedResource{
		{Type: payloads.Instance, Value: 1},
		{Type: payloads.MemMB, Value: wl.Requirements.MemMB},
		{Type: payloads.VCPUs, Value: wl.Requirements.VCPUs}}
	client.ctl.qs.Release(i.TenantID, resources...)
	return nil
}

func (client *ssntpClient) RemoveInstance(instanceID string) {
	err := client.releaseResources(instanceID)
	if err != nil {
		glog.Warningf("Error when releasing resources for deleted instance: %v", err)
	}
	client.deleteEphemeralStorage(instanceID)

	i, err := client.ctl.ds.GetInstance(instanceID)
	if err != nil {
		glog.Warningf("Error getting instance from datastore: %v", err)
		return
	}

	err = client.ctl.ds.DeleteInstance(instanceID)
	if err != nil {
		glog.Warningf("Error deleting instance from datastore: %v", err)
	}

	if i.CNCI {
		tenant, err := client.ctl.ds.GetTenant(i.TenantID)
		if err != nil {
			glog.Warningf("Error retrieving tenant %v", err)
			return
		}

		err = tenant.CNCIctrl.CNCIRemoved(i.ID)
		if err != nil {
			glog.Warningf("Error removing CNCI: %v", err)
		}
	}

	// notify anyone is listening for a state change
	err = i.TransitionInstanceState(payloads.Deleted)
	if err != nil {
		glog.Warningf("Error transitioning CNCI to deleted: %v", err)
	}
}

func (client *ssntpClient) instanceDeleted(payload []byte) {
	var event payloads.EventInstanceDeleted
	err := yaml.Unmarshal(payload, &event)
	if err != nil {
		glog.Warningf("Error unmarshalling InstanceDeleted: %v", err)
		return
	}
	client.RemoveInstance(event.InstanceDeleted.InstanceUUID)
}

func (client *ssntpClient) instanceStopped(payload []byte) {
	var event payloads.EventInstanceStopped
	err := yaml.Unmarshal(payload, &event)
	if err != nil {
		glog.Warning("Error unmarshalling InstanceStopped: %v")
		return
	}
	instanceID := event.InstanceStopped.InstanceUUID
	glog.Infof("Stopped instance %s", instanceID)

	i, err := client.ctl.ds.GetInstance(instanceID)
	if err != nil {
		glog.Warningf("Error getting instance from datastore: %v", err)
		return
	}

	err = client.ctl.ds.InstanceStopped(instanceID)
	if err != nil {
		glog.Warningf("Error stopping instance from datastore: %v", err)
	}

	if i.CNCI {
		tenant, err := client.ctl.ds.GetTenant(i.TenantID)
		if err != nil {
			glog.Warningf("Error retrieving tenant %v", err)
			return
		}
		err = tenant.CNCIctrl.CNCIStopped(i.ID)
		if err != nil {
			glog.Warningf("Error stopping CNCI: %v", err)
		}
	}
}

func (client *ssntpClient) concentratorInstanceAdded(payload []byte) {
	var event payloads.EventConcentratorInstanceAdded
	err := yaml.Unmarshal(payload, &event)
	if err != nil {
		glog.Warningf("Error unmarshalling EventConcentratorInstanceAdded: %v", err)
		return
	}
	newCNCI := event.CNCIAdded
	i, err := client.ctl.ds.GetInstance(newCNCI.InstanceUUID)
	if err != nil {
		glog.Warningf("Error getting instance: %v", err)
		return
	}

	i.IPAddress = newCNCI.ConcentratorIP
	i.MACAddress = newCNCI.ConcentratorMAC

	err = client.ctl.ds.UpdateInstance(i)
	if err != nil {
		glog.Warningf("Error updating CNCI Info: %v", err)
	}

	tenant, err := client.ctl.ds.GetTenant(i.TenantID)
	if err != nil || tenant == nil {
		glog.Warningf("Error getting tenant: %v", err)
		return
	}

	err = tenant.CNCIctrl.CNCIAdded(newCNCI.InstanceUUID)
	if err != nil {
		glog.Warningf("Error adding CNCI: %v", err)
	}
}

func (client *ssntpClient) traceReport(payload []byte) {
	var trace payloads.Trace
	err := yaml.Unmarshal(payload, &trace)
	if err != nil {
		glog.Warningf("Error unmarshalling TraceReport: %v", err)
		return
	}
	err = client.ctl.ds.HandleTraceReport(trace)
	if err != nil {
		glog.Warningf("Error updating trace report in datastore: %v", err)
	}
}

func (client *ssntpClient) nodeConnected(payload []byte) {
	var nodeConnected payloads.NodeConnected
	err := yaml.Unmarshal(payload, &nodeConnected)
	if err != nil {
		glog.Warningf("Error unmarshalling NodeConnected: %v", err)
		return
	}
	glog.Infof("Node %s connected", nodeConnected.Connected.NodeUUID)

	client.ctl.ds.AddNode(nodeConnected.Connected.NodeUUID, nodeConnected.Connected.NodeType)
}

func (client *ssntpClient) nodeDisconnected(payload []byte) {
	var nodeDisconnected payloads.NodeDisconnected
	err := yaml.Unmarshal(payload, &nodeDisconnected)
	if err != nil {
		glog.Warningf("Error unmarshalling NodeDisconnected: %v", err)
		return
	}

	glog.Infof("Node %s disconnected", nodeDisconnected.Disconnected.NodeUUID)
	err = client.ctl.ds.DeleteNode(nodeDisconnected.Disconnected.NodeUUID)
	if err != nil {
		glog.Warningf("Error marking node as deleted in datastore: %v", err)
	}
}

func (client *ssntpClient) unassignEvent(payload []byte) {
	var event payloads.EventPublicIPUnassigned
	err := yaml.Unmarshal(payload, &event)
	if err != nil {
		glog.Warningf("Error unmarshalling EventPublicIPUnassigned: %v", err)
		return
	}

	i, err := client.ctl.ds.GetInstance(event.UnassignedIP.InstanceUUID)
	if err != nil {
		glog.Warningf("Error getting instance from datastore: %v", err)
		return
	}

	err = client.ctl.ds.UnMapExternalIP(event.UnassignedIP.PublicIP)
	if err != nil {
		glog.Warningf("Error unmapping external IP: %v", err)
		return
	}

	client.ctl.qs.Release(i.TenantID, payloads.RequestedResource{Type: payloads.ExternalIP, Value: 1})

	msg := fmt.Sprintf("Unmapped %s from %s", event.UnassignedIP.PublicIP, event.UnassignedIP.PrivateIP)
	err = client.ctl.ds.LogEvent(i.TenantID, msg)
	if err != nil {
		glog.Warningf("Error logging event: %v", err)
	}
}

func (client *ssntpClient) assignEvent(payload []byte) {
	var event payloads.EventPublicIPAssigned
	err := yaml.Unmarshal(payload, &event)
	if err != nil {
		glog.Warningf("Error unmarshalling EventPublicIPAssigned: %v", err)
		return
	}

	i, err := client.ctl.ds.GetInstance(event.AssignedIP.InstanceUUID)
	if err != nil {
		glog.Warningf("Error getting instance from datastore: %v", err)
		return
	}

	msg := fmt.Sprintf("Mapped %s to %s", event.AssignedIP.PublicIP, event.AssignedIP.PrivateIP)
	err = client.ctl.ds.LogEvent(i.TenantID, msg)
	if err != nil {
		glog.Warningf("Error logging event: %v", err)
	}
}

func (client *ssntpClient) EventNotify(event ssntp.Event, frame *ssntp.Frame) {
	payload := frame.Payload

	glog.Info("EVENT ", event, " for ", client.name)

	glog.V(1).Info(string(payload))

	switch event {
	case ssntp.InstanceDeleted:
		client.instanceDeleted(payload)

	case ssntp.InstanceStopped:
		client.instanceStopped(payload)

	case ssntp.ConcentratorInstanceAdded:
		client.concentratorInstanceAdded(payload)

	case ssntp.TraceReport:
		client.traceReport(payload)

	case ssntp.NodeConnected:
		client.nodeConnected(payload)

	case ssntp.NodeDisconnected:
		client.nodeDisconnected(payload)

	case ssntp.PublicIPAssigned:
		client.assignEvent(payload)

	case ssntp.PublicIPUnassigned:
		client.unassignEvent(payload)

	}
}

func (client *ssntpClient) startFailure(payload []byte) {
	var failure payloads.ErrorStartFailure
	err := yaml.Unmarshal(payload, &failure)
	if err != nil {
		glog.Warningf("Error unmarshalling StartFailure: %v", err)
		return
	}
	if failure.Reason.IsFatal() && !failure.Restart {
		client.deleteEphemeralStorage(failure.InstanceUUID)
		err = client.releaseResources(failure.InstanceUUID)
		if err != nil {
			glog.Warningf("Error when releasing resources for start failed instance: %v", err)
		}
	}

	i, err := client.ctl.ds.GetInstance(failure.InstanceUUID)
	if err != nil {
		glog.Warningf("Error getting instance: %v", err)
		return
	}

	cnci := i.CNCI
	tenantID := i.TenantID

	err = client.ctl.ds.StartFailure(failure.InstanceUUID, failure.Reason, failure.Restart, failure.NodeUUID)
	if err != nil {
		glog.Warningf("Error adding StartFailure to datastore: %v", err)
	}

	if cnci {
		tenant, err := client.ctl.ds.GetTenant(tenantID)
		if err != nil {
			glog.Warningf("Unable to send start failure event: Error getting tenant %v", err)
			return
		}

		err = tenant.CNCIctrl.StartFailure(failure.InstanceUUID)
		if err != nil {
			glog.Warningf("Error adding StartFailure to datastore: %v", err)
		}
	}
}

func (client *ssntpClient) attachVolumeFailure(payload []byte) {
	var failure payloads.ErrorAttachVolumeFailure
	err := yaml.Unmarshal(payload, &failure)
	if err != nil {
		glog.Warningf("Error unmarshalling AttachVolumeFailure: %v", err)
		return
	}
	err = client.ctl.ds.AttachVolumeFailure(failure.InstanceUUID, failure.VolumeUUID, failure.Reason)
	if err != nil {
		glog.Warningf("Error handling AttachVolumeFailure in datastore: %v", err)
	}
}

func (client *ssntpClient) assignError(payload []byte) {
	var failure payloads.ErrorPublicIPFailure
	err := yaml.Unmarshal(payload, &failure)
	if err != nil {
		glog.Warningf("Error unmarshalling ErrorPublicIPFailure:: %v", err)
		return
	}

	err = client.ctl.ds.UnMapExternalIP(failure.PublicIP)
	if err != nil {
		glog.Warningf("Error unmapping external IP: %v", err)
	}

	client.ctl.qs.Release(failure.TenantUUID, payloads.RequestedResource{Type: payloads.ExternalIP, Value: 1})

	msg := fmt.Sprintf("Failed to map %s to %s: %s", failure.PublicIP, failure.InstanceUUID, failure.Reason.String())
	err = client.ctl.ds.LogError(failure.TenantUUID, msg)
	if err != nil {
		glog.Warningf("Error logging error: %v", err)
	}
}

func (client *ssntpClient) unassignError(payload []byte) {
	var failure payloads.ErrorPublicIPFailure
	err := yaml.Unmarshal(payload, &failure)
	if err != nil {
		glog.Warningf("Error unmarshalling ErrorPublicIPFailure: %v", err)
		return
	}

	// we can't unmap the IP - all we can do is log.
	msg := fmt.Sprintf("Failed to unmap %s from %s: %s", failure.PublicIP, failure.InstanceUUID, failure.Reason.String())
	err = client.ctl.ds.LogError(failure.TenantUUID, msg)
	if err != nil {
		glog.Warningf("Error logging error: %v", err)
	}
}

func (client *ssntpClient) ErrorNotify(err ssntp.Error, frame *ssntp.Frame) {
	payload := frame.Payload

	glog.Info("ERROR (", err, ") for ", client.name)
	glog.V(1).Info(string(payload))

	switch err {
	case ssntp.StartFailure:
		client.startFailure(payload)

	case ssntp.AttachVolumeFailure:
		client.attachVolumeFailure(payload)

	case ssntp.AssignPublicIPFailure:
		client.assignError(payload)

	case ssntp.UnassignPublicIPFailure:
		client.unassignError(payload)

	}
}

func newSSNTPClient(ctl *controller, config *ssntp.Config) (controllerClient, error) {
	client := &ssntpClient{name: "ciao Controller", ctl: ctl}

	err := client.ssntp.Dial(config, client)
	return client, err
}

func (client *ssntpClient) StartTracedWorkload(config string, startTime time.Time, label string) error {
	glog.V(1).Info("START TRACED config:")
	glog.V(1).Info(config)

	traceConfig := &ssntp.TraceConfig{
		PathTrace: true,
		Start:     startTime,
		Label:     []byte(label),
	}

	_, err := client.ssntp.SendTracedCommand(ssntp.START, []byte(config), traceConfig)

	return err
}

func (client *ssntpClient) StartWorkload(config string) error {
	glog.V(1).Info("START config:")
	glog.V(1).Info(config)

	_, err := client.ssntp.SendCommand(ssntp.START, []byte(config))

	return err
}

func (client *ssntpClient) deleteInstance(payload *payloads.Delete, instanceID string, nodeID string) error {
	y, err := yaml.Marshal(*payload)
	if err != nil {
		return err
	}

	glog.Info("DELETE instance_id: ", instanceID, "node_id ", nodeID)
	glog.V(1).Info(string(y))

	_, err = client.ssntp.SendCommand(ssntp.DELETE, y)

	return err
}

func (client *ssntpClient) DeleteInstance(instanceID string, nodeID string) error {
	if nodeID == "" {
		// This instance is not running and not assigned to a node.  We
		// can just remove its details from controller's db and delete
		// any ephemeral storage.
		glog.Info("Deleting unassigned instance")
		client.RemoveInstance(instanceID)
		return nil
	}

	payload := payloads.Delete{
		Delete: payloads.StopCmd{
			InstanceUUID:      instanceID,
			WorkloadAgentUUID: nodeID,
		},
	}

	return client.deleteInstance(&payload, instanceID, nodeID)
}

func (client *ssntpClient) StopInstance(instanceID string, nodeID string) error {
	payload := payloads.Delete{
		Delete: payloads.StopCmd{
			InstanceUUID:      instanceID,
			WorkloadAgentUUID: nodeID,
			Stop:              true,
		},
	}

	return client.deleteInstance(&payload, instanceID, nodeID)
}

func (client *ssntpClient) RestartInstance(i *types.Instance, w *types.Workload,
	t *types.Tenant) error {
	var cnci *types.Instance

	err := client.ctl.ds.InstanceRestarting(i.ID)
	if err != nil {
		return errors.Wrapf(err, "Unable to update instance state before restarting")
	}

	if !i.CNCI {
		// get the CNCI for this instance
		cnci, err = t.CNCIctrl.GetInstanceCNCI(i.ID)
		if err != nil {
			return err
		}
	}

	hostname := i.ID
	if i.Name != "" {
		hostname = i.Name
	}

	metaData := userData{
		UUID:     i.ID,
		Hostname: hostname,
	}

	attachments := client.ctl.ds.GetStorageAttachments(i.ID)

	restartCmd := payloads.StartCmd{
		TenantUUID:          i.TenantID,
		InstanceUUID:        i.ID,
		FWType:              payloads.Firmware(w.FWType),
		VMType:              w.VMType,
		InstancePersistence: payloads.Host,
		Requirements:        w.Requirements,
		Networking: payloads.NetworkResources{
			VnicMAC:  i.MACAddress,
			VnicUUID: i.VnicUUID,
		},
		Storage: make([]payloads.StorageResource, len(attachments)),
		Restart: true,
	}

	if cnci != nil {
		restartCmd.Networking.ConcentratorUUID = cnci.ID
		restartCmd.Networking.ConcentratorIP = cnci.IPAddress
		restartCmd.Networking.Subnet = i.Subnet
		restartCmd.Networking.PrivateIP = i.IPAddress
	}

	if w.VMType == payloads.Docker {
		restartCmd.DockerImage = w.ImageName
	}

	for k := range attachments {
		vol := &restartCmd.Storage[k]
		vol.ID = attachments[k].BlockID
		vol.Bootable = attachments[k].Boot
		vol.Ephemeral = attachments[k].Ephemeral
	}

	payload := payloads.Start{
		Start: restartCmd,
	}

	y, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	b, err := json.Marshal(&metaData)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	_, _ = buf.WriteString("---\n")
	_, _ = buf.Write(y)
	_, _ = buf.WriteString("...\n")
	_, _ = buf.WriteString(w.Config)
	_, _ = buf.WriteString("---\n")
	_, _ = buf.Write(b)
	_, _ = buf.WriteString("\n...\n")

	glog.Info("RESTART instance: ", i.ID)
	glog.V(1).Info(buf.String())

	_, err = client.ssntp.SendCommand(ssntp.START, buf.Bytes())

	return err
}

func (client *ssntpClient) EvacuateNode(nodeID string) error {
	evacuateCmd := payloads.EvacuateCmd{
		WorkloadAgentUUID: nodeID,
	}

	payload := payloads.Evacuate{
		Evacuate: evacuateCmd,
	}

	y, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	glog.Info("EVACUATE node: ", nodeID)
	glog.V(1).Info(string(y))

	_, err = client.ssntp.SendCommand(ssntp.EVACUATE, y)

	return err
}

func (client *ssntpClient) RestoreNode(nodeID string) error {
	restoreCmd := payloads.RestoreCmd{
		WorkloadAgentUUID: nodeID,
	}

	payload := payloads.Restore{
		Restore: restoreCmd,
	}

	y, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	glog.Info("Restore node: ", nodeID)
	glog.V(1).Info(string(y))

	_, err = client.ssntp.SendCommand(ssntp.Restore, y)

	return err
}

func (client *ssntpClient) attachVolume(volID string, instanceID string, nodeID string) error {
	payload := payloads.AttachVolume{
		Attach: payloads.VolumeCmd{
			InstanceUUID:      instanceID,
			VolumeUUID:        volID,
			WorkloadAgentUUID: nodeID,
		},
	}

	y, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	glog.Infof("AttachVolume %s to %s\n", volID, instanceID)
	glog.V(1).Info(string(y))

	_, err = client.ssntp.SendCommand(ssntp.AttachVolume, y)

	return err
}

func (client *ssntpClient) ssntpClient() *ssntp.Client {
	return &client.ssntp
}

func (client *ssntpClient) Disconnect() {
	client.ssntp.Close()
}

func (client *ssntpClient) mapExternalIP(t types.Tenant, m types.MappedIP) error {
	// get the CNCI for this instance
	i, err := t.CNCIctrl.GetInstanceCNCI(m.InstanceID)
	if err != nil {
		return err
	}

	payload := payloads.CommandAssignPublicIP{
		AssignIP: payloads.PublicIPCommand{
			ConcentratorUUID: i.ID,
			TenantUUID:       m.TenantID,
			InstanceUUID:     m.InstanceID,
			PublicIP:         m.ExternalIP,
			PrivateIP:        m.InternalIP,
			VnicMAC:          i.MACAddress,
		},
	}

	y, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	glog.Infof("Request Map of %s to %s\n", m.ExternalIP, m.InternalIP)
	glog.V(1).Info(string(y))

	_, err = client.ssntp.SendCommand(ssntp.AssignPublicIP, y)
	return err
}

func (client *ssntpClient) unMapExternalIP(t types.Tenant, m types.MappedIP) error {
	// get the CNCI for this instance
	i, err := t.CNCIctrl.GetInstanceCNCI(m.InstanceID)
	if err != nil {
		return err
	}

	payload := payloads.CommandReleasePublicIP{
		ReleaseIP: payloads.PublicIPCommand{
			ConcentratorUUID: i.ID,
			TenantUUID:       m.TenantID,
			InstanceUUID:     m.InstanceID,
			PublicIP:         m.ExternalIP,
			PrivateIP:        m.InternalIP,
			VnicMAC:          i.MACAddress,
		},
	}

	y, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	glog.Infof("Request unmap of %s from %s\n", m.ExternalIP, m.InternalIP)
	glog.V(1).Info(string(y))

	_, err = client.ssntp.SendCommand(ssntp.ReleasePublicIP, y)
	return err
}

func (client *ssntpClient) CNCIRefresh(cnciID string, cnciList []payloads.CNCINet) error {
	payload := payloads.CommandCNCIRefresh{
		Command: payloads.CNCIRefreshCommand{
			CNCIUUID: cnciID,
			CNCIList: cnciList,
		},
	}

	y, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	glog.Infof("Refresh CNCI %s: %v\n", cnciID, cnciList)
	glog.V(1).Info(string(y))

	_, err = client.ssntp.SendCommand(ssntp.RefreshCNCI, y)
	return err
}
