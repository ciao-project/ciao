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

type StartFailureReason string

const (
	FullCloud       StartFailureReason = "full_cloud"
	FullComputeNode                    = "full_cn"
	NoComputeNodes                     = "no_cn"
	NoNetworkNodes                     = "no_net_cn"
	InvalidPayload                     = "invalid_payload"
	InvalidData                        = "invalid_data"
	AlreadyRunning                     = "already_running"
	InstanceExists                     = "instance_exists"
	ImageFailure                       = "image_failure"
	LaunchFailure                      = "launch_failure"
	NetworkFailure                     = "network_failure"
)

type ErrorStartFailure struct {
	InstanceUUID string             `yaml:"instance_uuid"`
	Reason       StartFailureReason `yaml:"reason"`
}

func (s *ErrorStartFailure) Init() {
	s.InstanceUUID = ""
	s.Reason = ""
}

func (r StartFailureReason) String() string {
	switch r {
	case FullCloud:
		return "Cloud is full"
	case FullComputeNode:
		return "Compute node is full"
	case NoComputeNodes:
		return "No compute node available"
	case NoNetworkNodes:
		return "No network node available"
	case InvalidPayload:
		return "YAML payload is corrupt"
	case InvalidData:
		return "Command section of YAML payload is corrupt or missing required information"
	case AlreadyRunning:
		return "Instance is already running"
	case InstanceExists:
		return "Instance already exists"
	case ImageFailure:
		return "Failed to create instance image"
	case LaunchFailure:
		return "Failed to launch instance"
	case NetworkFailure:
		return "Failed to create VNIC for instance"
	}

	return ""
}
