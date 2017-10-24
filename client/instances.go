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
	"bytes"
	"fmt"
	"net/http"

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/pkg/errors"
)

// CreateInstances creates instances by the given request
func (client *Client) CreateInstances(request api.CreateServerRequest) (api.Servers, error) {
	var servers api.Servers

	url := client.buildCiaoURL("%s/instances", client.TenantID)
	err := client.postResource(url, api.InstancesV1, &request, &servers)

	return servers, err
}

// DeleteInstance deletes the given instance
func (client *Client) DeleteInstance(instanceID string) error {
	url := client.buildCiaoURL("%s/instances/%s", client.TenantID, instanceID)
	return client.deleteResource(url, api.InstancesV1)
}

func (client *Client) instanceAction(instanceID string, action string) error {
	actionBytes := []byte(action)

	url := client.buildCiaoURL("%s/instances/%s/action", client.TenantID, instanceID)

	resp, err := client.sendHTTPRequest("POST", url, nil, bytes.NewReader(actionBytes), api.InstancesV1)
	if err != nil {
		return errors.Wrap(err, "Error making HTTP request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("HTTP response code from %s not as expected: %d", url, resp.StatusCode)
	}
	return nil
}

// StopInstance stops the given instance
func (client *Client) StopInstance(instanceID string) error {
	return client.instanceAction(instanceID, "os-stop")
}

// StartInstance stops the given instance
func (client *Client) StartInstance(instanceID string) error {
	return client.instanceAction(instanceID, "os-start")
}

// ListInstancesByWorkload provides the list of instances for a given tenant and workloadID.
func (client *Client) ListInstancesByWorkload(tenantID string, workloadID string) (api.Servers, error) {
	var servers api.Servers

	url := client.buildCiaoURL("%s/instances/detail", tenantID)

	values := []queryValue{}
	if workloadID != "" {
		values = append(values, queryValue{
			name:  "workload",
			value: workloadID,
		})
	}

	err := client.getResource(url, api.InstancesV1, values, &servers)

	return servers, err

}

// ListInstances gets the set of instances
func (client *Client) ListInstances() (api.Servers, error) {
	return client.ListInstancesByWorkload(client.TenantID, "")
}

// GetInstance gets the details of a single instances
func (client *Client) GetInstance(instanceID string) (api.Server, error) {
	var server api.Server

	url := client.buildCiaoURL("%s/instances/%s", client.TenantID, instanceID)
	err := client.getResource(url, api.InstancesV1, nil, &server)

	return server, err
}
