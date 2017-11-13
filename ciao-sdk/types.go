package sdk

import (
	"github.com/ciao-project/ciao/ciao-controller/types"
)

type source struct {
	Type types.SourceType `yaml:"service"`
	ID   string           `yaml:"id"`
}

type disk struct {
	ID        *string `yaml:"volume_id,omitempty"`
	Size      int     `yaml:"size"`
	Bootable  bool    `yaml:"bootable"`
	Source    source  `yaml:"source"`
	Ephemeral bool    `yaml:"ephemeral"`
}

type defaultResources struct {
	VCPUs int `yaml:"vcpus"`
	MemMB int `yaml:"mem_mb"`
}

// Workload contains detailed information about a workload
type Workload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	CPUs int    `json:"vcpus"`
	Mem  int    `json:"ram"`
}

// we currently only use the first disk due to lack of support
// in types.Workload for multiple storage resources.
type workloadOptions struct {
	Description     string           `yaml:"description"`
	VMType          string           `yaml:"vm_type"`
	FWType          string           `yaml:"fw_type,omitempty"`
	ImageName       string           `yaml:"image_name,omitempty"`
	Defaults        defaultResources `yaml:"defaults"`
	CloudConfigFile string           `yaml:"cloud_init,omitempty"`
	Disks           []disk           `yaml:"disks,omitempty"`
}

type Project struct {
	ID   string `mapstructure:"id"`
	Name string `mapstructure:"name"`
}

type CommandOpts struct {
	Args        []string
	TenantID    string
	ComputeName string
	All         bool
	CNCINode    bool
	ComputeNode bool
	NetworkNode bool
	Detail      bool
	Quotas      bool
	Resources   bool
	Config      bool
	Limit       int
	Marker      string
	Offset      int
	Tenant      string
	Workload    string
}
