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

type TenantAddedEvent struct {
	AgentUUID        string `yaml:"agent_uuid"`
	AgentIP          string `yaml:"agent_ip"`
	TenantUUID       string `yaml:"tenant_uuid"`
	TenantSubnet     string `yaml:"tenant_subnet"`
	ConcentratorUUID string `yaml:"concentrator_uuid"`
	ConcentratorIP   string `yaml:"concentrator_ip"`
	SubnetKey        int    `yaml:"subnet_key"`
}

type EventTenantAdded struct {
	TenantAdded TenantAddedEvent `yaml:"tenant_added"`
}

type EventTenantRemoved struct {
	TenantRemoved TenantAddedEvent `yaml:"tenant_removed"`
}
