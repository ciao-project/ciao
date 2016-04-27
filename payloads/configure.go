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

type ServiceType string
type StorageType string

const (
	Glance   ServiceType = "glance"
	Keystone ServiceType = "keystone"
)

const (
	Filesystem StorageType = "file"
	Etcd       StorageType = "etcd"
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

type ConfigureScheduler struct {
	ConfigStorageType StorageType `yaml:"storage_type"`
	ConfigStorageURI  string      `yaml:"storage_uri"`
}

type ConfigureController struct {
	ComputePort      int    `yaml:"compute_port"`
	ComputeCACert    string `yaml:"compute_ca"`
	ComputeCert      string `yaml:"compute_cert"`
	IdentityUser     string `yaml:"identity_user"`
	IdentityPassword string `yaml:"identity_password"`
}

type ConfigureLauncher struct {
	ComputeNetwork    string `yaml:"compute_net"`
	ManagementNetwork string `yaml:"mgmt_net"`
	DiskLimit         bool   `yaml:"disk_limit"`
	MemoryLimit       bool   `yaml:"mem_limit"`
}

type ConfigureService struct {
	Type ServiceType `yaml:"type"`
	URL  string      `yaml:"url"`
}

type ConfigurePayload struct {
	Scheduler       ConfigureScheduler  `yaml:"scheduler"`
	Controller      ConfigureController `yaml:"controller"`
	Launcher        ConfigureLauncher   `yaml:"launcher"`
	ImageService    ConfigureService    `yaml:"image_service"`
	IdentityService ConfigureService    `yaml:"identity_service"`
}

type Configure struct {
	Configure ConfigurePayload `yaml:"configure"`
}
