//
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
//

package testutil

import (
	"github.com/01org/ciao/ssntp"
)

// CmdResult is a common result structure for tests spanning between
// controller client, scheduler server, and the various (eg: Agent,
// NetAgent, CNCIAgent) agent roles.
type CmdResult struct {
	InstanceUUID string
	Err          error
	NodeUUID     string
	TenantUUID   string
	CNCI         bool
}

// RoleToTestCert returns a string containing the testutil certificate
// matching the specified ssntp.Role
func RoleToTestCert(role ssntp.Role) string {
	switch role {
	case ssntp.SCHEDULER:
		return TestCertScheduler
	case ssntp.SERVER:
		return TestCertServer
	case ssntp.AGENT:
		return TestCertAgent
	case ssntp.Controller:
		return TestCertController
	case ssntp.CNCIAGENT:
		return TestCertCNCIAgent
	case ssntp.NETAGENT:
		return TestCertNetAgent
	case ssntp.AGENT | ssntp.NETAGENT:
		return TestCertAgentNetAgent
	}

	return TestCertUnknown
}
