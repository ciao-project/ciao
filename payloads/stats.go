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

type InstanceStat struct {
	InstanceUUID string `yaml:"instance_uuid"`
	State        string `yaml:"state"`

	// IP address to use to connect to instance via SSH.  This
	// is actually the IP address of the CNCI VM.
	// Will be "" if the instance is itself a CNCI VM.
	SSHIP string `yaml:"ssh_ip"`

	// Port number used to access the SSH service running on the
	// VM.  This number is computed from the VM's IP address.
	// Will be 0 if the instance is itself a CNCI VM.
	SSHPort int `yaml:"ssh_port"`

	// Memory usage in MB.  May be -1 if State != Running.
	MemoryUsageMB int `yaml:"memory_usage_mb"`

	// Disk usage in MB.  May be -1 if State = Pending.
	DiskUsageMB int `yaml:"disk_usage_mb"`

	// Percentage of CPU Usage for VM, normalized for VCPUs.
	// May be -1 if State != Running or if launcher has not
	// acquired enough samples to compute the CPU usage.
	// Assuming CPU usage can be computed it will be a value
	// between 0 and 100% regardless of the number of VPCUs.
	// 100% means all your VCPUs are maxed out.
	CPUUsage int `yaml:"cpu_usage"`
}

type NetworkStat struct {
	NodeIP  string `yaml:"ip"`
	NodeMAC string `yaml:"mac"`
}

type Stat struct {
	NodeUUID        string `yaml:"node_uuid"`
	Status          string `yaml:"status"`
	MemTotalMB      int    `yaml:"mem_total_mb"`
	MemAvailableMB  int    `yaml:"mem_available_mb"`
	DiskTotalMB     int    `yaml:"disk_total_mb"`
	DiskAvailableMB int    `yaml:"disk_available_mb"`
	Load            int    `yaml:"load"`
	CpusOnline      int    `yaml:"cpus_online"`
	NodeHostName    string `yaml:"hostname"`
	Networks        []NetworkStat
	Instances       []InstanceStat
}

const (
	Pending    = "pending"
	Running    = "running"
	Exited     = "exited"
	ExitFailed = "exit_failed"
	ExitPaused = "exit_paused"
)

func (s *Stat) Init() {
	s.NodeUUID = ""
	s.MemTotalMB = -1
	s.MemAvailableMB = -1
	s.DiskTotalMB = -1
	s.DiskAvailableMB = -1
	s.Load = -1
	s.CpusOnline = -1
}
