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

type StopFailureReason string

const (
	StopNoInstance     StopFailureReason = "no_instance"
	StopInvalidPayload                   = "invalid_payload"
	StopInvalidData                      = "invalid_data"
	StopAlreadyStopped                   = "already_stopped"
)

type ErrorStopFailure struct {
	InstanceUUID string            `yaml:"instance_uuid"`
	Reason       StopFailureReason `yaml:"reason"`
}

func (s *ErrorStopFailure) Init() {
	s.InstanceUUID = ""
	s.Reason = ""
}

func (r StopFailureReason) String() string {
	switch r {
	case StopNoInstance:
		return "Instance does not exist"
	case StopInvalidPayload:
		return "YAML payload is corrupt"
	case StopInvalidData:
		return "Command section of YAML payload is corrupt or missing required information"
	case StopAlreadyStopped:
		return "Instance has already shut down"
	}

	return ""
}
