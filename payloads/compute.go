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

// PrivateAddresses contains information about a single instance network
// interface.
type PrivateAddresses struct {
	Addr               string `json:"addr"`
	OSEXTIPSMACMacAddr string `json:"OS-EXT-IPS-MAC:mac_addr"`
	OSEXTIPSType       string `json:"OS-EXT-IPS:type"`
	Version            int    `json:"version"`
}

// Addresses contains information about an instance's networks.
type Addresses struct {
	Private []PrivateAddresses `json:"private"`
}

// Link is reserved for future use.
type Link struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

// Flavor identifies the flavour (workload) of an instance.
type Flavor struct {
	ID    string `json:"id"`
	Links []Link `json:"links"`
}

// Image identifies the base image of the instance.
type Image struct {
	ID    string `json:"id"`
	Links []Link `json:"links"`
}

// SecurityGroup represents the security group of an instance.
type SecurityGroup struct {
	Name string `json:"name"`
}

const (
	// ComputeStatusPending is a filter that used to select pending
	// instances in requests to the controller.
	ComputeStatusPending = "pending"

	// ComputeStatusRunning is a filter that used to select running
	// instances in requests to the controller.
	ComputeStatusRunning = "running"

	// ComputeStatusStopped is a filter that used to select exited
	// instances in requests to the controller.
	ComputeStatusStopped = "exited"
)

// Server contains information about a specific instance within a ciao cluster.
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

// ComputeServers represents the unmarshalled version of the contents of a
// /v2.1/{tenant}/servers/detail response.  It contains information about a
// set of instances within a ciao cluster.
type ComputeServers struct {
	TotalServers int      `json:"total_servers"`
	Servers      []Server `json:"servers"`
}

// NewComputeServers allocates a ComputeServers structure.
// It allocates the Servers slice as well so that the marshalled
// JSON is an empty array and not a nil pointer for, as
// specified by the OpenStack APIs.
func NewComputeServers() (servers ComputeServers) {
	servers.Servers = []Server{}
	return
}

// ComputeServer represents the unmarshalled version of the contents of a
// /v2.1/{tenant}/servers/{server} response.  It contains information about a
// specific instance within a ciao cluster.
type ComputeServer struct {
	Server Server `json:"server"`
}

// ComputeFlavors represents the unmarshalled version of the contents of a
// /v2.1/{tenant}/flavors response.  It contains information about all the
// flavors in a cluster.
type ComputeFlavors struct {
	Flavors []struct {
		ID    string `json:"id"`
		Links []Link `json:"links"`
		Name  string `json:"name"`
	} `json:"flavors"`
}

// NewComputeFlavors allocates a ComputeFlavors structure.
// It allocates the Flavors slice as well so that the marshalled
// JSON is an empty array and not a nil pointer, as specified
// by the OpenStack APIs.
func NewComputeFlavors() (flavors ComputeFlavors) {
	flavors.Flavors = []struct {
		ID    string `json:"id"`
		Links []Link `json:"links"`
		Name  string `json:"name"`
	}{}
	return
}

// FlavorDetails contains information about a specific flavor.
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

// ComputeFlavorDetails represents the unmarshalled version of the contents of a
// /v2.1/{tenant}/flavors/{flavor} response.  It contains information about a
// specific flavour.
type ComputeFlavorDetails struct {
	Flavor FlavorDetails `json:"flavor"`
}

// ComputeFlavorsDetails represents the unmarshalled version of the contents of a
// /v2.1/{tenant}/flavors/detail response. It contains detailed information about
// all flavour for a given tenant.
type ComputeFlavorsDetails struct {
	Flavors []FlavorDetails `json:"flavors"`
}

// NewComputeFlavorsDetails allocates a ComputeFlavorsDetails structure.
// It allocates the Flavors slice as well so that the marshalled
// JSON is an empty array and not a nil pointer, as specified by the
// OpenStack APIs.
func NewComputeFlavorsDetails() (flavors ComputeFlavorsDetails) {
	flavors.Flavors = []FlavorDetails{}
	return
}

// ComputeCreateServer represents the unmarshalled version of the contents of a
// /v2.1/{tenant}/servers request.  It contains the information needed to start
// one or more instances.
type ComputeCreateServer struct {
	Server struct {
		Name         string `json:"name"`
		Image        string `json:"imageRef"`
		Workload     string `json:"flavorRef"`
		MaxInstances int    `json:"max_count"`
		MinInstances int    `json:"min_count"`
	} `json:"server"`
}

// CiaoComputeTenants represents the unmarshalled version of the contents of a
// /v2.1/tenants response.  It contains information about the tenants in a ciao
// cluster.
type CiaoComputeTenants struct {
	Tenants []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"tenants"`
}

// NewCiaoComputeTenants allocates a CiaoComputeTenants structure.
// It allocates the Tenants slice as well so that the marshalled
// JSON is an empty array and not a nil pointer, as specified by the
// OpenStack APIs.
func NewCiaoComputeTenants() (tenants CiaoComputeTenants) {
	tenants.Tenants = []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{}
	return
}

// CiaoComputeNode contains status and statistic information for an individual
// node.
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

// CiaoComputeNodes represents the unmarshalled version of the contents of a
// /v2.1/nodes response.  It contains status and statistics information
// for a set of nodes.
type CiaoComputeNodes struct {
	Nodes []CiaoComputeNode `json:"nodes"`
}

// NewCiaoComputeNodes allocates a CiaoComputeNodes structure.
// It allocates the Nodes slice as well so that the marshalled
// JSON is an empty array and not a nil pointer, as specified by the
// OpenStack APIs.
func NewCiaoComputeNodes() (nodes CiaoComputeNodes) {
	nodes.Nodes = []CiaoComputeNode{}
	return
}

// CiaoTenantResources represents the unmarshalled version of the contents of a
// /v2.1/{tenant}/quotas response.  It contains the current resource usage
// information for a tenant.
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

// CiaoUsage contains a snapshot of resource consumption for a tenant.
type CiaoUsage struct {
	VCPU      int       `json:"cpus_usage"`
	Memory    int       `json:"ram_usage"`
	Disk      int       `json:"disk_usage"`
	Timestamp time.Time `json:"timestamp"`
}

// CiaoUsageHistory represents the unmarshalled version of the contents of a
// /v2.1/{tenant}/resources response.  It contains snapshots of usage information
// for a given tenant over a given period of time.
type CiaoUsageHistory struct {
	Usages []CiaoUsage `json:"usage"`
}

// CiaoCNCISubnet contains subnet information for a CNCI.
type CiaoCNCISubnet struct {
	Subnet string `json:"subnet_cidr"`
}

// CiaoCNCI contains information about an individual CNCI.
type CiaoCNCI struct {
	ID        string           `json:"id"`
	TenantID  string           `json:"tenant_id"`
	IPv4      string           `json:"IPv4"`
	Geography string           `json:"geography"`
	Subnets   []CiaoCNCISubnet `json:"subnets"`
}

// CiaoCNCIDetail represents the unmarshalled version of the contents of a
// v2.1/cncis/{cnci}/detail response.  It contains information about a CNCI.
type CiaoCNCIDetail struct {
	CiaoCNCI `json:"cnci"`
}

// CiaoCNCIs represents the unmarshalled version of the contents of a
// v2.1/cncis response.  It contains information about all the CNCIs
// in the ciao cluster.
type CiaoCNCIs struct {
	CNCIs []CiaoCNCI `json:"cncis"`
}

// NewCiaoCNCIs allocates a CiaoCNCIs structure.
// It allocates the CNCIs slice as well so that the marshalled
// JSON is an empty array and not a nil pointer, as specified by the
// OpenStack APIs.
func NewCiaoCNCIs() (cncis CiaoCNCIs) {
	cncis.CNCIs = []CiaoCNCI{}
	return
}

// CiaoServerStats contains status information about a CN or a NN.
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

// CiaoServersStats represents the unmarshalled version of the contents of a
// v2.1/nodes/{node}/servers/detail response.  It contains general information
// about a group of instances.
type CiaoServersStats struct {
	TotalServers int               `json:"total_servers"`
	Servers      []CiaoServerStats `json:"servers"`
}

// NewCiaoServersStats allocates a CiaoServersStats structure.
// It allocates the Servers slice as well so that the marshalled
// JSON is an empty array and not a nil pointer, as specified by the
// OpenStack APIs.
func NewCiaoServersStats() (servers CiaoServersStats) {
	servers.Servers = []CiaoServerStats{}
	return
}

// CiaoClusterStatus represents the unmarshalled version of the contents of a
// v2.1/nodes/summary response.  It contains information about the nodes that
// make up a ciao cluster.
type CiaoClusterStatus struct {
	Status struct {
		TotalNodes            int `json:"total_nodes"`
		TotalNodesReady       int `json:"total_nodes_ready"`
		TotalNodesFull        int `json:"total_nodes_full"`
		TotalNodesOffline     int `json:"total_nodes_offline"`
		TotalNodesMaintenance int `json:"total_nodes_maintenance"`
	} `json:"cluster"`
}

// CNCIDetail is reserved for future use.
type CNCIDetail struct {
	IPv4 string `json:"IPv4"`
}

// CiaoServersAction represents the unmarshalled version of the contents of a
// v2.1/servers/action request.  It contains an action to be performed on
// one or more instances.
type CiaoServersAction struct {
	Action    string   `json:"action"`
	ServerIDs []string `json:"servers"`
}

// CiaoTraceSummary contains information about a specific SSNTP Trace label.
type CiaoTraceSummary struct {
	Label     string `json:"label"`
	Instances int    `json:"instances"`
}

// CiaoTracesSummary represents the unmarshalled version of the response to a
// v2.1/traces request.  It contains a list of all trace labels and the
// number of instances associated with them.
type CiaoTracesSummary struct {
	Summaries []CiaoTraceSummary `json:"summaries"`
}

// CiaoFrameStat is reserved for future use
type CiaoFrameStat struct {
	ID               string  `json:"node_id"`
	TotalElapsedTime float64 `json:"total_elapsed_time"`
	ControllerTime   float64 `json:"total_controller_time"`
	LauncherTime     float64 `json:"total_launcher_time"`
	SchedulerTime    float64 `json:"total_scheduler_time"`
}

// CiaoBatchFrameStat contains frame statisitics for a ciao cluster.
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

// CiaoTraceData represents the unmarshalled version of the response to a
// v2.1/traces/{label} request.  It contains statistics computed from the trace
// information of SSNTP commands sent within a ciao cluster.
type CiaoTraceData struct {
	Summary    CiaoBatchFrameStat `json:"summary"`
	FramesStat []CiaoFrameStat    `json:"frames"`
}

// CiaoEvent contains information about an individual event generated
// in a ciao cluster.
type CiaoEvent struct {
	Timestamp time.Time `json:"time_stamp"`
	TenantID  string    `json:"tenant_id"`
	EventType string    `json:"type"`
	Message   string    `json:"message"`
}

// CiaoEvents represents the unmarshalled version of the response to a
// v2.1/{tenant}/event or v2.1/event request.
type CiaoEvents struct {
	Events []CiaoEvent `json:"events"`
}

// NewCiaoEvents allocates a CiaoEvents structure.
// It allocates the Events slice as well so that the marshalled
// JSON is an empty array and not a nil pointer, as specified by the
// OpenStack APIs.
func NewCiaoEvents() (events CiaoEvents) {
	events.Events = []CiaoEvent{}
	return
}
