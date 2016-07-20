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

package types

import (
	"time"

	"github.com/01org/ciao/ciao-storage"
	"github.com/01org/ciao/payloads"
)

// SourceType contains the valid values of the storage source.
type SourceType string

const (
	// ImageService indicates the source comes from the image service.
	ImageService SourceType = "image"

	// VolumeService indicates the source comes from the volume service.
	VolumeService SourceType = "volume"
)

// StorageResource defines a storage resource for a workload.
// TBD: should the workload support multiple of these?
type StorageResource struct {
	// ID indicates a volumeID. If ID is blank, then it needs to be created.
	ID string

	// Bootable indicates whether should the resource be used for booting
	Bootable bool

	// Persistent indicates whether the storage is temporary
	// TBD: do we bother to save info about temp storage?
	//      does it count against quota?
	Persistent bool

	// Size is the size of the storage to be created if new.
	Size int

	// ImageType indicates whether we are making a new resource
	// based on an image or existing volume.
	// Needed only for new storage.
	SourceType SourceType
}

// Workload contains resource and configuration information for a user
// workload.
type Workload struct {
	ID          string                       `json:"id"`
	Description string                       `json:"description"`
	FWType      string                       `json:"-"`
	VMType      payloads.Hypervisor          `json:"-"`
	ImageID     string                       `json:"-"`
	ImageName   string                       `json:"-"`
	Config      string                       `json:"-"`
	Defaults    []payloads.RequestedResource `json:"-"`
	Storage     *StorageResource             `json:"-"`
}

// Instance contains information about an instance of a workload.
type Instance struct {
	ID         string         `json:"instance_id"`
	TenantID   string         `json:"tenant_id"`
	State      string         `json:"instance_state"`
	WorkloadID string         `json:"workload_id"`
	NodeID     string         `json:"node_id"`
	MACAddress string         `json:"mac_address"`
	IPAddress  string         `json:"ip_address"`
	SSHIP      string         `json:"ssh_ip"`
	SSHPort    int            `json:"ssh_port"`
	CNCI       bool           `json:"-"`
	Usage      map[string]int `json:"-"`
}

// SortedInstancesByID implements sort.Interface for Instance by ID string
type SortedInstancesByID []*Instance

func (s SortedInstancesByID) Len() int           { return len(s) }
func (s SortedInstancesByID) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s SortedInstancesByID) Less(i, j int) bool { return s[i].ID < s[j].ID }

// SortedComputeNodesByID implements sort.Interface for Node by ID string
type SortedComputeNodesByID []payloads.CiaoComputeNode

func (s SortedComputeNodesByID) Len() int           { return len(s) }
func (s SortedComputeNodesByID) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s SortedComputeNodesByID) Less(i, j int) bool { return s[i].ID < s[j].ID }

// Tenant contains information about a tenant or project.
type Tenant struct {
	ID        string
	Name      string
	CNCIID    string
	CNCIMAC   string
	CNCIIP    string
	Resources []*Resource
}

// Resource contains quota or limit information on a resource type.
type Resource struct {
	Rname string
	Rtype int
	Limit int
	Usage int
}

// OverLimit calculates whether a request will put a tenant over it's limit.
func (r *Resource) OverLimit(request int) bool {
	if r.Limit > 0 && r.Usage+request > r.Limit {
		return true
	}
	return false
}

// LogEntry stores information about events.
type LogEntry struct {
	Timestamp time.Time `json:"time_stamp"`
	TenantID  string    `json:"tenant_id"`
	EventType string    `json:"type"`
	Message   string    `json:"message"`
}

// NodeStats stores statistics for individual nodes in the cluster.
type NodeStats struct {
	NodeID          string    `json:"node_id"`
	Timestamp       time.Time `json:"time_stamp"`
	Load            int       `json:"load"`
	MemTotalMB      int       `json:"mem_total_mb"`
	MemAvailableMB  int       `json:"mem_available_mb"`
	DiskTotalMB     int       `json:"mem_total_mb"`
	DiskAvailableMB int       `json:"disk_available_mb"`
	CpusOnline      int       `json:"cpus_online"`
}

// NodeSummary contains summary information for all nodes in the cluster.
type NodeSummary struct {
	NodeID                string `json:"node_id"`
	TotalInstances        int    `json:"total_instances"`
	TotalRunningInstances int    `json:"total_running_instances"`
	TotalPendingInstances int    `json:"total_pending_instances"`
	TotalPausedInstances  int    `json:"total_paused_instances"`
}

// TenantCNCI contains information about the CNCI instance for a tenant.
type TenantCNCI struct {
	TenantID   string   `json:"tenant_id"`
	IPAddress  string   `json:"ip_address"`
	MACAddress string   `json:"mac_address"`
	InstanceID string   `json:"instance_id"`
	Subnets    []string `json:"subnets"`
}

// FrameStat contains tracing information per node.
type FrameStat struct {
	ID               string  `json:"node_id"`
	TotalElapsedTime float64 `json:"total_elapsed_time"`
	ControllerTime   float64 `json:"total_controller_time"`
	LauncherTime     float64 `json:"total_launcher_time"`
	SchedulerTime    float64 `json:"total_scheduler_time"`
}

// BatchFrameStat contains tracing information for a group of start requests
// by label.
type BatchFrameStat struct {
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

// BatchFrameSummary provides summary information on tracing per label.
type BatchFrameSummary struct {
	BatchID      string `json:"batch_id"`
	NumInstances int    `json:"num_instances"`
}

// Node contains information about a physical node in the cluster.
type Node struct {
	ID       string `json:"node_id"`
	IPAddr   string `json:"ip_address"`
	Hostname string `json:"hostname"`
}

// BlockState represents the state of the block device in the controller
// datastore.
type BlockState string

// BlockData respresents the attributes of this block device.
// TBD - do we really need to store this as actual data,
// or can we use a set of interfaces to get the info?
type BlockData struct {
	storage.BlockDevice
	TenantID   string     // the tenant who owns this volume
	Instances  []Instance // any instances using this volume
	Size       int        // size in GB
	State      BlockState // status of
	CreateTime time.Time  // when we created the volume
}
