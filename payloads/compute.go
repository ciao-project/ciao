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

import (
	"time"
)

type PrivateAddresses struct {
	Addr               string `json:"addr"`
	OSEXTIPSMACMacAddr string `json:"OS-EXT-IPS-MAC:mac_addr"`
	OSEXTIPSType       string `json:"OS-EXT-IPS:type"`
	Version            int    `json:"version"`
}

type Addresses struct {
	Private []PrivateAddresses `json:"private"`
}

type Link struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

type Flavor struct {
	ID    string `json:"id"`
	Links []Link `json:"links"`
}

type Image struct {
	ID    string `json:"id"`
	Links []Link `json:"links"`
}

type SecurityGroup struct {
	Name string `json:"name"`
}

const (
	ComputeStatusPending = "pending"
	ComputeStatusRunning = "running"
	ComputeStatusStopped = "exited"
)

type Server struct {
	Addresses                        Addresses       `json:"addresses"`
	Created                          time.Time       `json:"created"`
	Flavor                           Flavor          `json:"flavor"`
	HostID                           string          `json:"hostId"`
	ID                               string          `json:"id"`
	Image                            Image           `json:"image"`
	KeyName                          string          `json:"key_name"`
	Links                            []Link          `json:"links"`
	Name                             string          `json:"name"`
	AccessIPv4                       string          `json:"accessIPv4"`
	AccessIPv6                       string          `json:"accessIPv6"`
	ConfigDrive                      string          `json:"config_drive"`
	OSDCFDiskConfig                  string          `json:"OS-DCF:diskConfig"`
	OSEXTAZAvailabilityZone          string          `json:"OS-EXT-AZ:availability_zone"`
	OSEXTSRVATTRHost                 string          `json:"OS-EXT-SRV-ATTR:host"`
	OSEXTSRVATTRHypervisorHostname   string          `json:"OS-EXT-SRV-ATTR:hypervisor_hostname"`
	OSEXTSRVATTRInstanceName         string          `json:"OS-EXT-SRV-ATTR:instance_name"`
	OSEXTSTSPowerState               int             `json:"OS-EXT-STS:power_state"`
	OSEXTSTSTaskState                string          `json:"OS-EXT-STS:task_state"`
	OSEXTSTSVMState                  string          `json:"OS-EXT-STS:vm_state"`
	OsExtendedVolumesVolumesAttached []string        `json:"os-extended-volumes:volumes_attached"`
	OSSRVUSGLaunchedAt               time.Time       `json:"OS-SRV-USG:launched_at"`
	OSSRVUSGTerminatedAt             time.Time       `json:"OS-SRV-USG:terminated_at"`
	Progress                         int             `json:"progress"`
	SecurityGroups                   []SecurityGroup `json:"security_groups"`
	Status                           string          `json:"status"`
	HostStatus                       string          `json:"host_status"`
	TenantID                         string          `json:"tenant_id"`
	Updated                          time.Time       `json:"updated"`
	UserID                           string          `json:"user_id"`
	SSHIP                            string          `json:"ssh_ip"`
	SSHPort                          int             `json:"ssh_port"`
}

type ComputeServers struct {
	TotalServers int      `json:"total_servers"`
	Servers      []Server `json:"servers"`
}

type ComputeServer struct {
	Server Server `json:"server"`
}

type ComputeFlavors struct {
	Flavors []struct {
		ID    string `json:"id"`
		Links []Link `json:"links"`
		Name  string `json:"name"`
	} `json:"flavors"`
}

type FlavorDetails struct {
	OSFLVDISABLEDDisabled  bool   `json:"OS-FLV-DISABLED:disabled"`
	Disk                   string `json:"disk"` /* OpenStack API says this is an int */
	OSFLVEXTDATAEphemeral  int    `json:"OS-FLV-EXT-DATA:ephemeral"`
	OsFlavorAccessIsPublic bool   `json:"os-flavor-access:is_public"`
	ID                     string `json:"id"`
	Links                  []Link `json:"links"`
	Name                   string `json:"name"`
	RAM                    int    `json:"ram"`
	Swap                   string `json:"swap"`
	Vcpus                  int    `json:"vcpus"`
}

type ComputeFlavorDetails struct {
	Flavor FlavorDetails `json:"flavor"`
}

type ComputeCreateServer struct {
	Server struct {
		Name         string `json:"name"`
		Image        string `json:"imageRef"`
		Workload     string `json:"flavorRef"`
		MaxInstances int    `json:"max_count"`
		MinInstances int    `json:"min_count"`
	} `json:"server"`
}

type CiaoComputeTenants struct {
	Tenants []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"tenants"`
}

type CiaoComputeNode struct {
	ID                    string    `json:"id"`
	Timestamp             time.Time `json:"updated"`
	Status                string    `json:"status"`
	MemTotal              int       `json:"ram_total"`
	MemAvailable          int       `json:"ram_available"`
	DiskTotal             int       `json:"disk_total"`
	DiskAvailable         int       `json:"disk_available"`
	Load                  int       `json:"load"`
	OnlineCPUs            int       `json:"online_cpus"`
	TotalInstances        int       `json:"total_instances"`
	TotalRunningInstances int       `json:"total_running_instances"`
	TotalPendingInstances int       `json:"total_pending_instances"`
	TotalPausedInstances  int       `json:"total_paused_instances"`
}

type CiaoComputeNodes struct {
	Nodes []CiaoComputeNode `json:"nodes"`
}

type CiaoTenantResources struct {
	ID            string    `json:"id"`
	Timestamp     time.Time `json:"updated"`
	InstanceLimit int       `json:"instances_limit"`
	InstanceUsage int       `json:"instances_usage"`
	VCPULimit     int       `json:"cpus_limit"`
	VCPUUsage     int       `json:"cpus_usage"`
	MemLimit      int       `json:"ram_limit"`
	MemUsage      int       `json:"ram_usage"`
	DiskLimit     int       `json:"disk_limit"`
	DiskUsage     int       `json:"disk_usage"`
}

type CiaoUsage struct {
	VCPU      int       `json:"cpus_usage"`
	Memory    int       `json:"ram_usage"`
	Disk      int       `json:"disk_usage"`
	Timestamp time.Time `json:"timestamp"`
}

type CiaoUsageHistory struct {
	Usages []CiaoUsage `json: usage`
}

type CiaoCNCISubnet struct {
	Subnet string `json:"subnet_cidr"`
}

type CiaoCNCI struct {
	ID        string           `json:"id"`
	TenantID  string           `json:"tenant_id"`
	IPv4      string           `json:"IPv4"`
	Geography string           `json:"geography"`
	Subnets   []CiaoCNCISubnet `json:"subnets"`
}

type CiaoCNCIDetail struct {
	CiaoCNCI `json:"cnci"`
}

type CiaoCNCIs struct {
	CNCIs []CiaoCNCI `json:"cncis"`
}

type CiaoServerStats struct {
	ID        string    `json:"id"`
	NodeID    string    `json:"node_id"`
	Timestamp time.Time `json:"updated"`
	Status    string    `json:"status"`
	TenantID  string    `json:"tenant_id"`
	IPv4      string    `json:"IPv4"`
	VCPUUsage int       `json:"cpus_usage"`
	MemUsage  int       `json:"ram_usage"`
	DiskUsage int       `json:"disk_usage"`
}

type CiaoServersStats struct {
	TotalServers int               `json:"total_servers"`
	Servers      []CiaoServerStats `json:"servers"`
}

type CiaoClusterStatus struct {
	Status struct {
		TotalNodes            int `json:"total_nodes"`
		TotalNodesReady       int `json:"total_nodes_ready"`
		TotalNodesFull        int `json:"total_nodes_full"`
		TotalNodesOffline     int `json:"total_nodes_offline"`
		TotalNodesMaintenance int `json:"total_nodes_maintenance"`
	} `json:"cluster"`
}

type CNCIDetail struct {
	IPv4 string `json:"IPv4"`
}

type CiaoServersAction struct {
	Action    string   `json:"action"`
	ServerIDs []string `json:"servers"`
}

type CiaoTraceSummary struct {
	Label     string `json:"label"`
	Instances int    `json:"instances"`
}

type CiaoTracesSummary struct {
	Summaries []CiaoTraceSummary `json:"summaries"`
}

type CiaoFrameStat struct {
	ID               string  `json:"node_id"`
	TotalElapsedTime float64 `json:"total_elapsed_time"`
	ControllerTime   float64 `json:"total_controller_time"`
	LauncherTime     float64 `json:"total_launcher_time"`
	SchedulerTime    float64 `json:"total_scheduler_time"`
}

type CiaoBatchFrameStat struct {
	NumInstances             int     `json:"num_instances"`
	TotalElapsed             float64 `json:"total_elapsed"`
	AverageElapsed           float64 `json:"average_elapsed"`
	AverageControllerElapsed float64 `json:"average_controller_elapsed"`
	AverageLauncherElapsed   float64 `json:"average_launcher_elapsed"`
	AverageSchedulerElapsed  float64 `json:"average_scheduler_elapsed"`
	VarianceController       float64 `json:"controller_variance"`
	VarianceLauncher         float64 `json:"launcher_variance"`
	VarianceScheduler        float64 `json:"scheduler_variance"`
}

type CiaoTraceData struct {
	Summary    CiaoBatchFrameStat `json:"summary"`
	FramesStat []CiaoFrameStat    `json:"frames"`
}

type CiaoEvent struct {
	Timestamp time.Time `json:"time_stamp"`
	TenantId  string    `json:"tenant_id"`
	EventType string    `json:"type"`
	Message   string    `json:"message"`
}

type CiaoEvents struct {
	Events []CiaoEvent `json:"events"`
}
