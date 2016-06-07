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

// ServiceType is reserved for future use.
type ServiceType string

// StorageType is reserved for future use.
type StorageType string

const (
	// Glance is reserved for future use.
	Glance ServiceType = "glance"

	// Keystone is reserved for future use.
	Keystone ServiceType = "keystone"
)

const (
	// Filesystem is reserved for future use.
	Filesystem StorageType = "file"

	// Etcd is reserved for future use.
	Etcd StorageType = "etcd"
)

func (s ServiceType) String() string {
	switch s {
	case Glance:
		return "glance"
	case Keystone:
		return "keystone"
	}

	return ""
}

func (s StorageType) String() string {
	switch s {
	case Filesystem:
		return "file"
	case Etcd:
		return "etcd"
	}

	return ""
}

// ConfigureScheduler is reserved for future use.
type ConfigureScheduler struct {
	ConfigStorageType StorageType `yaml:"storage_type"`
	ConfigStorageURI  string      `yaml:"storage_uri"`
}

// ConfigureController is reserved for future use.
type ConfigureController struct {
	ComputePort      int    `yaml:"compute_port"`
	HTTPSCACert      string `yaml:"compute_ca"`
	HTTPSKey         string `yaml:"compute_cert"`
	IdentityUser     string `yaml:"identity_user"`
	IdentityPassword string `yaml:"identity_password"`
}

// ConfigureLauncher is reserved for future use.
type ConfigureLauncher struct {
	ComputeNetwork    string `yaml:"compute_net"`
	ManagementNetwork string `yaml:"mgmt_net"`
	DiskLimit         bool   `yaml:"disk_limit"`
	MemoryLimit       bool   `yaml:"mem_limit"`
}

// ConfigureService is reserved for future use.
type ConfigureService struct {
	Type ServiceType `yaml:"type"`
	URL  string      `yaml:"url"`
}

// ConfigurePayload is reserved for future use.
type ConfigurePayload struct {
	Scheduler       ConfigureScheduler  `yaml:"scheduler"`
	Controller      ConfigureController `yaml:"controller"`
	Launcher        ConfigureLauncher   `yaml:"launcher"`
	ImageService    ConfigureService    `yaml:"image_service"`
	IdentityService ConfigureService    `yaml:"identity_service"`
}

// Configure is reserved for future use.
type Configure struct {
	Configure ConfigurePayload `yaml:"configure"`
}

// InitDefaults initializes default vaulues for Configure structure.
func (conf *Configure) InitDefaults() {
	conf.Configure.Scheduler.ConfigStorageType = Filesystem
	conf.Configure.Controller.ComputePort = 8774
	conf.Configure.ImageService.Type = Glance
	conf.Configure.IdentityService.Type = Keystone
	conf.Configure.Launcher.DiskLimit = true
	conf.Configure.Launcher.MemoryLimit = true
}
