// Copyright (c) 2017 Intel Corporation
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

package payloads

// CNCINet contains information about CNCI network interfaces.
type CNCINet struct {
	PhysicalIP string `yaml:"physical_ip"`
	Subnet     string `yaml:"subnet"`
	TunnelIP   string `yaml:"tunnel_ip"`
	TunnelID   uint32 `yaml:"tunnel_id"`
}

// CNCIRefreshCommand contains information on where to send
// the updated concentrator instance list.
type CNCIRefreshCommand struct {
	CNCIUUID string    `yaml:"cnci_uuid"`
	CNCIList []CNCINet `yaml:"cncis"`
}

// CommandCNCIRefresh represents the unmarshalled version of the
// contents of an SSNTP ssntp.ConcentratorInstanceRefresh command. This command
// is sent by the controller to the cnci-agent when new cncis are added or an
// existing cnci is modified.
type CommandCNCIRefresh struct {
	Command CNCIRefreshCommand `yaml:"cnci_refresh"`
}
