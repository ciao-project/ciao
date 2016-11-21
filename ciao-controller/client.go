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
	"fmt"
	"time"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

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
			glog.Warning("error unmarshalling temp stat")
			return
		}
		client.ctl.ds.HandleStats(stats)
	}
	glog.V(1).Info(string(payload))
}

func (client *ssntpClient) deleteEphemeralStorage(instanceID string) {
	attachments, _ := client.ctl.ds.GetStorageAttachments(instanceID)
	for _, attachment := range attachments {
		if !attachment.Ephemeral {
			continue
		}
		err := client.ctl.ds.DeleteStorageAttachment(attachment.ID)
		if err != nil {
			glog.Warningf("Error deleting attachment from datastore: %v", err)
		}
		err = client.ctl.ds.DeleteBlockDevice(attachment.BlockID)
		if err != nil {
			glog.Warningf("Error deleting block device from datastore: %v", err)
		}
		err = client.ctl.DeleteBlockDevice(attachment.BlockID)
		if err != nil {
			glog.Warningf("Error deleting block device: %v", err)
		}
	}
}

func (client *ssntpClient) EventNotify(event ssntp.Event, frame *ssntp.Frame) {
	payload := frame.Payload

	glog.Info("EVENT ", event, " for ", client.name)

	glog.V(1).Info(string(payload))

	switch event {
	case ssntp.InstanceDeleted:
		var event payloads.EventInstanceDeleted
		err := yaml.Unmarshal(payload, &event)
		if err != nil {
			glog.Warning("Error unmarshalling InstanceDeleted")
			return
		}
		client.deleteEphemeralStorage(event.InstanceDeleted.InstanceUUID)
		client.ctl.ds.DeleteInstance(event.InstanceDeleted.InstanceUUID)
	case ssntp.ConcentratorInstanceAdded:
		var event payloads.EventConcentratorInstanceAdded
		err := yaml.Unmarshal(payload, &event)
		if err != nil {
			glog.Warning(err)
			return
		}
		newCNCI := event.CNCIAdded
		client.ctl.ds.AddCNCIIP(newCNCI.ConcentratorMAC, newCNCI.ConcentratorIP)
	case ssntp.TraceReport:
		var trace payloads.Trace
		err := yaml.Unmarshal(payload, &trace)
		if err != nil {
			glog.Warning("error unmarshalling TraceReport")
			return
		}
		client.ctl.ds.HandleTraceReport(trace)

	case ssntp.NodeConnected:
		var nodeConnected payloads.NodeConnected
		err := yaml.Unmarshal(payload, &nodeConnected)
		if err != nil {
			glog.Warning("error unmarshalling NodeConnected")
			return
		}
		glog.Infof("Node %s connected", nodeConnected.Connected.NodeUUID)

	case ssntp.NodeDisconnected:
		var nodeDisconnected payloads.NodeDisconnected
		err := yaml.Unmarshal(payload, &nodeDisconnected)
		if err != nil {
			glog.Warning("error unmarshalling NodeDisconnected")
			return
		}

		glog.Infof("Node %s disconnected", nodeDisconnected.Disconnected.NodeUUID)
		client.ctl.ds.DeleteNode(nodeDisconnected.Disconnected.NodeUUID)

	case ssntp.PublicIPAssigned:
		var event payloads.EventPublicIPAssigned
		err := yaml.Unmarshal(payload, &event)
		if err != nil {
			glog.Warning(err)
			return
		}

		i, err := client.ctl.ds.GetInstance(event.AssignedIP.InstanceUUID)
		if err != nil {
			glog.Warning(err)
			return
		}

		msg := fmt.Sprintf("Mapped %s to %s", event.AssignedIP.PublicIP, event.AssignedIP.PrivateIP)
		client.ctl.ds.LogEvent(i.TenantID, msg)

	case ssntp.PublicIPUnassigned:
		var event payloads.EventPublicIPUnassigned
		err := yaml.Unmarshal(payload, &event)
		if err != nil {
			glog.Warning(err)
			return
		}

		i, err := client.ctl.ds.GetInstance(event.UnassignedIP.InstanceUUID)
		if err != nil {
			glog.Warning(err)
			return
		}

		err = client.ctl.ds.UnMapExternalIP(event.UnassignedIP.PublicIP)
		if err != nil {
			glog.Warning(err)
			return
		}

		msg := fmt.Sprintf("Unmapped %s from %s", event.UnassignedIP.PublicIP, event.UnassignedIP.PrivateIP)
		client.ctl.ds.LogEvent(i.TenantID, msg)
	}
}

func (client *ssntpClient) ErrorNotify(err ssntp.Error, frame *ssntp.Frame) {
	payload := frame.Payload

	glog.Info("ERROR (", err, ") for ", client.name)
	glog.V(1).Info(string(payload))

	switch err {
	case ssntp.StartFailure:
		var failure payloads.ErrorStartFailure
		err := yaml.Unmarshal(payload, &failure)
		if err != nil {
			glog.Warning("Error unmarshalling StartFailure")
			return
		}
		client.ctl.ds.StartFailure(failure.InstanceUUID, failure.Reason)
	case ssntp.StopFailure:
		var failure payloads.ErrorStopFailure
		err := yaml.Unmarshal(payload, &failure)
		if err != nil {
			glog.Warning("Error unmarshalling StopFailure")
			return
		}
		client.ctl.ds.StopFailure(failure.InstanceUUID, failure.Reason)
	case ssntp.RestartFailure:
		var failure payloads.ErrorRestartFailure
		err := yaml.Unmarshal(payload, &failure)
		if err != nil {
			glog.Warning("Error unmarshalling RestartFailure")
			return
		}
		client.ctl.ds.RestartFailure(failure.InstanceUUID, failure.Reason)
	case ssntp.AttachVolumeFailure:
		var failure payloads.ErrorAttachVolumeFailure
		err := yaml.Unmarshal(payload, &failure)
		if err != nil {
			glog.Warning("Error unmarshalling AttachVolumeFailure")
			return
		}
		client.ctl.ds.AttachVolumeFailure(failure.InstanceUUID, failure.VolumeUUID, failure.Reason)

	case ssntp.DetachVolumeFailure:
		var failure payloads.ErrorDetachVolumeFailure
		err := yaml.Unmarshal(payload, &failure)
		if err != nil {
			glog.Warning("Error unmarshalling DetachVolumeFailure")
			return
		}
		client.ctl.ds.DetachVolumeFailure(failure.InstanceUUID, failure.VolumeUUID, failure.Reason)

	case ssntp.AssignPublicIPFailure:
		var failure payloads.ErrorPublicIPFailure
		err := yaml.Unmarshal(payload, &failure)
		if err != nil {
			glog.Warning("Error unmarshalling ErrorPublicIPFailure")
			return
		}

		err = client.ctl.ds.UnMapExternalIP(failure.PublicIP)
		if err != nil {
			glog.Warning(err)
			return
		}

		msg := fmt.Sprintf("Failed to map %s to %s: %s", failure.PublicIP, failure.InstanceUUID, failure.Reason.String())
		client.ctl.ds.LogEvent(failure.TenantUUID, msg)

	}
}

func newSSNTPClient(ctl *controller, config *ssntp.Config) (*ssntpClient, error) {
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

func (client *ssntpClient) DeleteInstance(instanceID string, nodeID string) error {
	stopCmd := payloads.StopCmd{
		InstanceUUID:      instanceID,
		WorkloadAgentUUID: nodeID,
	}

	payload := payloads.Delete{
		Delete: stopCmd,
	}

	y, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	glog.Info("DELETE instance_id: ", instanceID, "node_id ", nodeID)
	glog.V(1).Info(string(y))

	_, err = client.ssntp.SendCommand(ssntp.DELETE, y)

	return err
}

func (client *ssntpClient) StopInstance(instanceID string, nodeID string) error {
	stopCmd := payloads.StopCmd{
		InstanceUUID:      instanceID,
		WorkloadAgentUUID: nodeID,
	}

	payload := payloads.Stop{
		Stop: stopCmd,
	}

	y, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	glog.Info("STOP instance_id: ", instanceID, "node_id ", nodeID)
	glog.V(1).Info(string(y))

	_, err = client.ssntp.SendCommand(ssntp.STOP, y)

	return err
}

func (client *ssntpClient) RestartInstance(instanceID string, nodeID string) error {
	restartCmd := payloads.RestartCmd{
		InstanceUUID:      instanceID,
		WorkloadAgentUUID: nodeID,
	}

	payload := payloads.Restart{
		Restart: restartCmd,
	}

	y, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	glog.Info("RESTART instance: ", instanceID)
	glog.V(1).Info(string(y))

	_, err = client.ssntp.SendCommand(ssntp.RESTART, y)

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

func (client *ssntpClient) detachVolume(volID string, instanceID string, nodeID string) error {
	payload := payloads.DetachVolume{
		Detach: payloads.VolumeCmd{
			InstanceUUID:      instanceID,
			VolumeUUID:        volID,
			WorkloadAgentUUID: nodeID,
		},
	}

	y, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	glog.Infof("DetachVolume %s to %s\n", volID, instanceID)
	glog.V(1).Info(string(y))

	_, err = client.ssntp.SendCommand(ssntp.DetachVolume, y)

	return err
}

func (client *ssntpClient) Disconnect() {
	client.ssntp.Close()
}

func (client *ssntpClient) mapExternalIP(t types.Tenant, m types.MappedIP) error {
	payload := payloads.CommandAssignPublicIP{
		AssignIP: payloads.PublicIPCommand{
			ConcentratorUUID: t.CNCIID,
			TenantUUID:       m.TenantID,
			InstanceUUID:     m.InstanceID,
			PublicIP:         m.ExternalIP,
			PrivateIP:        m.InternalIP,
			VnicMAC:          t.CNCIMAC,
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
	payload := payloads.CommandReleasePublicIP{
		ReleaseIP: payloads.PublicIPCommand{
			ConcentratorUUID: t.CNCIID,
			TenantUUID:       m.TenantID,
			InstanceUUID:     m.InstanceID,
			PublicIP:         m.ExternalIP,
			PrivateIP:        m.InternalIP,
			VnicMAC:          t.CNCIMAC,
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
