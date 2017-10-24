//
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
//

package client

import (
	"time"

	"github.com/ciao-project/ciao/ciao-controller/types"
)

// ListEvents retrieves the events for either all or the desired tenant
func (client *Client) ListEvents(tenantID string) (types.CiaoEvents, error) {
	var events types.CiaoEvents
	var url string

	if tenantID == "" {
		url = client.buildComputeURL("events")
	} else {
		url = client.buildComputeURL("%s/events", tenantID)
	}

	err := client.getResource(url, "", nil, &events)

	return events, err
}

// DeleteEvents deletes all events
func (client *Client) DeleteEvents() error {
	url := client.buildComputeURL("events")

	return client.deleteResource(url, "")
}

// ListInstancesByNode gets the instances on a given node
func (client *Client) ListInstancesByNode(nodeID string) (types.CiaoServersStats, error) {
	var servers types.CiaoServersStats

	url := client.buildComputeURL("nodes/%s/servers/detail", nodeID)
	err := client.getResource(url, "", nil, &servers)

	return servers, err
}

// DeleteAllInstances deletes all the instances
func (client *Client) DeleteAllInstances() error {
	var action types.CiaoServersAction

	url := client.buildComputeURL("%s/servers/action", client.TenantID)
	action.Action = "os-delete"

	return client.postResource(url, "", &action, nil)
}

// ListComputeNodes returns the set of compute nodes
func (client *Client) ListComputeNodes() (types.CiaoNodes, error) {
	var nodes types.CiaoNodes

	url := client.buildComputeURL("nodes/compute")
	err := client.getResource(url, "", nil, &nodes)

	return nodes, err
}

// ListNetworkNodes returns the set of network nodes
func (client *Client) ListNetworkNodes() (types.CiaoNodes, error) {
	var nodes types.CiaoNodes

	url := client.buildComputeURL("nodes/network")
	err := client.getResource(url, "", nil, &nodes)

	return nodes, err
}

// ListNodes returns the set of nodes
func (client *Client) ListNodes() (types.CiaoNodes, error) {
	var nodes types.CiaoNodes

	url := client.buildComputeURL("nodes")
	err := client.getResource(url, "", nil, &nodes)

	return nodes, err
}

// ListCNCIs returns the set of CNCIs
func (client *Client) ListCNCIs() (types.CiaoCNCIs, error) {
	var nodes types.CiaoCNCIs

	url := client.buildComputeURL("cncis")
	err := client.getResource(url, "", nil, &nodes)

	return nodes, err
}

// GetNodeSummary returns summary information about the cluster
func (client *Client) GetNodeSummary() (types.CiaoClusterStatus, error) {
	var status types.CiaoClusterStatus

	url := client.buildComputeURL("nodes/summary")
	err := client.getResource(url, "", nil, &status)

	return status, err
}

// GetCNCI returns information about a CNCI
func (client *Client) GetCNCI(cnciID string) (types.CiaoCNCI, error) {
	var cnci types.CiaoCNCI

	url := client.buildComputeURL("cncis/%s/detail", cnciID)
	err := client.getResource(url, "", nil, &cnci)

	return cnci, err
}

// ListTenantQuotas gets legacy tenant quota information
func (client *Client) ListTenantQuotas() (types.CiaoTenantResources, error) {
	var resources types.CiaoTenantResources

	url := client.buildComputeURL("%s/quotas", client.TenantID)
	err := client.getResource(url, "", nil, &resources)

	return resources, err
}

// ListTenantResources gets tenant usage information
func (client *Client) ListTenantResources() (types.CiaoUsageHistory, error) {
	var usage types.CiaoUsageHistory
	url := client.buildComputeURL("%s/resources", client.TenantID)

	now := time.Now()
	values := []queryValue{
		{
			name:  "start_date",
			value: now.Add(-15 * time.Minute).Format(time.RFC3339),
		},
		{
			name:  "end_date",
			value: now.Format(time.RFC3339),
		},
	}

	err := client.getResource(url, "", values, &usage)

	return usage, err
}

// ListTraceLabels returns a list of trace labels
func (client *Client) ListTraceLabels() (types.CiaoTracesSummary, error) {
	var traces types.CiaoTracesSummary

	url := client.buildComputeURL("traces")
	err := client.getResource(url, "", nil, &traces)

	return traces, err
}

// GetTraceData returns trace details
func (client *Client) GetTraceData(label string) (types.CiaoTraceData, error) {
	var data types.CiaoTraceData

	url := client.buildComputeURL("traces/%s", label)
	err := client.getResource(url, "", nil, &data)

	return data, err
}
