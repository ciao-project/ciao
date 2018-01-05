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

// StorageType is used to define the configuration backend storage type.
type StorageType string

const (
	// Filesystem defines the local filesystem backend storage type for the
	// configuration data.
	Filesystem StorageType = "file"
)

func (s StorageType) String() string {
	switch s {
	case Filesystem:
		return "file"
	}

	return ""
}

// ConfigureScheduler contains the unmarshalled configurations for the
// scheduler service.
type ConfigureScheduler struct {
	ConfigStorageURI string `yaml:"storage_uri"`
}

// ConfigureController contains the unmarshalled configurations for the
// controller service.
type ConfigureController struct {
	CiaoPort             int    `yaml:"ciao_port"`
	HTTPSCACert          string `yaml:"compute_ca"`
	HTTPSKey             string `yaml:"compute_cert"`
	CNCIVcpus            int    `yaml:"cnci_vcpus"`
	CNCIMem              int    `yaml:"cnci_mem"`
	CNCIDisk             int    `yaml:"cnci_disk"`
	AdminSSHKey          string `yaml:"admin_ssh_key"`
	ClientAuthCACertPath string `yaml:"client_auth_ca_cert_path"`
	CNCINet              string `yaml:"cnci_net"`
}

// ConfigureLauncher contains the unmarshalled configurations for the
// launcher service.
type ConfigureLauncher struct {
	ComputeNetwork    []string `yaml:"compute_net"`
	ManagementNetwork []string `yaml:"mgmt_net"`
	DiskLimit         bool     `yaml:"disk_limit"`
	MemoryLimit       bool     `yaml:"mem_limit"`
	ChildUser         string   `yaml:"child_user"`
}

// ConfigureStorage contains the unmarshalled configurations for the
// Ceph storage driver.
type ConfigureStorage struct {
	CephID string `yaml:"ceph_id"`
}

// ConfigurePayload is a wrapper to read and unmarshall all posible
// configurations for the following services: scheduler, controller, launcher,
//  imaging and identity.
type ConfigurePayload struct {
	Scheduler  ConfigureScheduler  `yaml:"scheduler"`
	Storage    ConfigureStorage    `yaml:"storage"`
	Controller ConfigureController `yaml:"controller"`
	Launcher   ConfigureLauncher   `yaml:"launcher"`
}

// Configure represents the SSNTP CONFIGURE command payload.
type Configure struct {
	Configure ConfigurePayload `yaml:"configure"`
}

// InitDefaults initializes default vaulues for Configure structure.
func (conf *Configure) InitDefaults() {
	conf.Configure.Controller.CiaoPort = 8889
	conf.Configure.Launcher.DiskLimit = true
	conf.Configure.Launcher.MemoryLimit = true
	conf.Configure.Controller.CNCIDisk = 2048
	conf.Configure.Controller.CNCIMem = 2048
	conf.Configure.Controller.CNCIVcpus = 4
	conf.Configure.Controller.CNCINet = "192.168.0.0"
}
