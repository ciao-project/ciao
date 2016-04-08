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

package payloads

type Persistence string
type Firmware string
type Resource string
type Hypervisor string

const (
	All  Persistence = "all"
	VM               = "vm"
	Host             = "host"
)

const (
	EFI    Firmware = "efi"
	Legacy          = "legacy"
)

const (
	VCPUs       Resource = "vcpus"
	MemMB                = "mem_mb"
	DiskMB               = "disk_mb"
	NetworkNode          = "network_node"
)

const (
	QEMU   Hypervisor = "qemu"
	Docker            = "docker"
)

type RequestedResource struct {
	Type      Resource `yaml:"type"`
	Value     int      `yaml:"value"`
	Mandatory bool     `yaml:"mandatory"`
}

type EstimatedResource struct {
	Type  Resource `yaml:"type"`
	Value int      `yaml:"value"`
}

type NetworkResources struct {
	VnicMAC          string `yaml:"vnic_mac"`
	VnicUUID         string `yaml:"vnic_uuid"`
	ConcentratorUUID string `yaml:"concentrator_uuid"`
	ConcentratorIP   string `yaml:"concentrator_ip"`
	Subnet           string `yaml:"subnet"`
	SubnetKey        string `yaml:"subnet_key"`
	SubnetUUID       string `yaml:"subnet_uuid"`
	PrivateIP        string `yaml:"private_ip"`
	PublicIP         bool   `yaml:"public_ip"`
}

type StartCmd struct {
	TenantUUID          string              `yaml:"tenant_uuid"`
	InstanceUUID        string              `yaml:"instance_uuid"`
	ImageUUID           string              `yaml:"image_uuid"`
	DockerImage         string              `yaml:"docker_image"`
	FWType              Firmware            `yaml:"fw_type"`
	InstancePersistence Persistence         `yaml:"persistence"`
	VMType              Hypervisor          `yaml:"vm_type"`
	RequestedResources  []RequestedResource `yaml:"requested_resources"`
	EstimatedResources  []EstimatedResource `yaml:"estimated_resources"`
	Networking          NetworkResources    `yaml:"networking"`
}

type Start struct {
	Start StartCmd `yaml:"start"`
}

type RestartCmd struct {
	TenantUUID          string              `yaml:"tenant_uuid"`
	InstanceUUID        string              `yaml:"instance_uuid"`
	ImageUUID           string              `yaml:"image_uuid"`
	WorkloadAgentUUID   string              `yaml:"workload_agent_uuid"`
	FWType              Firmware            `yaml:"fw_type"`
	InstancePersistence Persistence         `yaml:"persistence"`
	RequestedResources  []RequestedResource `yaml:"requested_resources"`
	EstimatedResources  []EstimatedResource `yaml:"estimated_resources"`
	Networking          NetworkResources    `yaml:"networking"`
}

type Restart struct {
	Restart RestartCmd `yaml:"restart"`
}
