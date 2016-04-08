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

const (
	Glance   ServiceType = "glance"
	Keystone ServiceType = "keystone"
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

type ConfigureService struct {
	Type ServiceType `yaml:"type"`
	URL  string      `yaml:"url"`
}

type ConfigureCmd struct {
	ImageService    ConfigureService `yaml: image_service"`
	IdentityService ConfigureService `yaml: identity_service"`
}

type CommandConfigure struct {
	Configure ConfigureCmd `yaml:"configure"`
}
