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

package payloads_test

import (
	"testing"

	. "github.com/01org/ciao/payloads"
	"github.com/docker/distribution/uuid"
	"gopkg.in/yaml.v2"
)

const instancePublicIP = "10.1.2.3"
const instancePrivateIP = "192.168.1.2"
const vnicMAC = "aa:bb:cc:01:02:03"

const assignIPYaml = "" +
	"assign_public_ip:\n" +
	"  concentrator_uuid: " + cnciUUID + "\n" +
	"  tenant_uuid: " + tenantUUID + "\n" +
	"  instance_uuid: " + instanceUUID + "\n" +
	"  public_ip: " + instancePublicIP + "\n" +
	"  private_ip: " + instancePrivateIP + "\n" +
	"  vnic_mac: " + vnicMAC + "\n"

const releaseIPYaml = "" +
	"release_public_ip:\n" +
	"  concentrator_uuid: " + cnciUUID + "\n" +
	"  tenant_uuid: " + tenantUUID + "\n" +
	"  instance_uuid: " + instanceUUID + "\n" +
	"  public_ip: " + instancePublicIP + "\n" +
	"  private_ip: " + instancePrivateIP + "\n" +
	"  vnic_mac: " + vnicMAC + "\n"

func TestAssignPublicIPUnmarshal(t *testing.T) {
	var assignIP CommandAssignPublicIP

	err := yaml.Unmarshal([]byte(assignIPYaml), &assignIP)
	if err != nil {
		t.Error(err)
	}

	if assignIP.AssignIP.ConcentratorUUID != cnciUUID {
		t.Errorf("Wrong concentrator UUID field [%s]", assignIP.AssignIP.ConcentratorUUID)
	}

	if assignIP.AssignIP.TenantUUID != tenantUUID {
		t.Errorf("Wrong tenant UUID field [%s]", assignIP.AssignIP.TenantUUID)
	}

	if assignIP.AssignIP.InstanceUUID != instanceUUID {
		t.Errorf("Wrong instance UUID field [%s]", assignIP.AssignIP.InstanceUUID)
	}

	if assignIP.AssignIP.PublicIP != instancePublicIP {
		t.Errorf("Wrong public IP field [%s]", assignIP.AssignIP.PublicIP)
	}

	if assignIP.AssignIP.PrivateIP != instancePrivateIP {
		t.Errorf("Wrong private IP field [%s]", assignIP.AssignIP.PrivateIP)
	}

	if assignIP.AssignIP.VnicMAC != vnicMAC {
		t.Errorf("Wrong VNIC MAC field [%s]", assignIP.AssignIP.VnicMAC)
	}
}

func TestReleasePublicIPUnmarshal(t *testing.T) {
	var releaseIP CommandReleasePublicIP

	err := yaml.Unmarshal([]byte(releaseIPYaml), &releaseIP)
	if err != nil {
		t.Error(err)
	}

	if releaseIP.ReleaseIP.ConcentratorUUID != cnciUUID {
		t.Errorf("Wrong concentrator UUID field [%s]", releaseIP.ReleaseIP.ConcentratorUUID)
	}

	if releaseIP.ReleaseIP.TenantUUID != tenantUUID {
		t.Errorf("Wrong tenant UUID field [%s]", releaseIP.ReleaseIP.TenantUUID)
	}

	if releaseIP.ReleaseIP.InstanceUUID != instanceUUID {
		t.Errorf("Wrong instance UUID field [%s]", releaseIP.ReleaseIP.InstanceUUID)
	}

	if releaseIP.ReleaseIP.PublicIP != instancePublicIP {
		t.Errorf("Wrong public IP field [%s]", releaseIP.ReleaseIP.PublicIP)
	}

	if releaseIP.ReleaseIP.PrivateIP != instancePrivateIP {
		t.Errorf("Wrong private IP field [%s]", releaseIP.ReleaseIP.PrivateIP)
	}

	if releaseIP.ReleaseIP.VnicMAC != vnicMAC {
		t.Errorf("Wrong VNIC MAC field [%s]", releaseIP.ReleaseIP.VnicMAC)
	}
}

func TestAssignPublicIPMarshal(t *testing.T) {
	var assignIP CommandAssignPublicIP

	assignIP.AssignIP.ConcentratorUUID = cnciUUID
	assignIP.AssignIP.TenantUUID = tenantUUID
	assignIP.AssignIP.InstanceUUID = instanceUUID
	assignIP.AssignIP.PublicIP = instancePublicIP
	assignIP.AssignIP.PrivateIP = instancePrivateIP
	assignIP.AssignIP.VnicMAC = vnicMAC

	y, err := yaml.Marshal(&assignIP)
	if err != nil {
		t.Error(err)
	}

	if string(y) != assignIPYaml {
		t.Errorf("AssignPublicIP marshalling failed\n[%s]\n vs\n[%s]", string(y), assignIPYaml)
	}
}

func TestReleasePublicIPMarshal(t *testing.T) {
	var releaseIP CommandReleasePublicIP

	releaseIP.ReleaseIP.ConcentratorUUID = cnciUUID
	releaseIP.ReleaseIP.TenantUUID = tenantUUID
	releaseIP.ReleaseIP.InstanceUUID = instanceUUID
	releaseIP.ReleaseIP.PublicIP = instancePublicIP
	releaseIP.ReleaseIP.PrivateIP = instancePrivateIP
	releaseIP.ReleaseIP.VnicMAC = vnicMAC

	y, err := yaml.Marshal(&releaseIP)
	if err != nil {
		t.Error(err)
	}

	if string(y) != releaseIPYaml {
		t.Errorf("ReleasePublicIP marshalling failed\n[%s]\n vs\n[%s]", string(y), releaseIPYaml)
	}
}

func TestPublicIPFailureString(t *testing.T) {
	var stringTests = []struct {
		r        PublicIPFailureReason
		expected string
	}{
		{PublicIPNoInstance, "Instance does not exist"},
		{PublicIPInvalidPayload, "YAML payload is corrupt"},
		{PublicIPInvalidData, "Command section of YAML payload is corrupt or missing required information"},
		{PublicIPAssignFailure, "Public IP assignment operation_failed"},
		{PublicIPReleaseFailure, "Public IP release operation_failed"},
	}
	error := ErrorPublicIPFailure{
		ConcentratorUUID: uuid.Generate().String(),
		TenantUUID:       uuid.Generate().String(),
		InstanceUUID:     uuid.Generate().String(),
		PublicIP:         "10.1.2.3",
		PrivateIP:        "192.168.1.2",
		VnicMAC:          "aa:bb:cc:01:02:03",
	}
	for _, test := range stringTests {
		error.Reason = test.r
		s := error.Reason.String()
		if s != test.expected {
			t.Errorf("expected \"%s\", got \"%s\"", test.expected, s)
		}
	}
}
