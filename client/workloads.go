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
	"fmt"

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/pkg/errors"
)

func (client *Client) getCiaoWorkloadsResource() (string, error) {
	return client.getCiaoResource("workloads", api.WorkloadsV1)
}

// ListWorkloads gets the workloads available
func (client *Client) ListWorkloads() ([]types.Workload, error) {
	var wls []types.Workload

	var url string
	if client.IsPrivileged() {
		url = client.buildCiaoURL("workloads")
	} else {
		url = client.buildCiaoURL("%s/workloads", client.TenantID)
	}

	err := client.getResource(url, api.WorkloadsV1, nil, &wls)
	return wls, err
}

// CreateWorkload creates a worklaod
func (client *Client) CreateWorkload(request types.Workload) (types.Workload, error) {
	url, err := client.getCiaoWorkloadsResource()
	if err != nil {
		return types.Workload{}, errors.Wrap(err, "Error getting workloads resource")
	}

	var response types.WorkloadResponse

	err = client.postResource(url, api.WorkloadsV1, &request, &response)

	return response.Workload, err
}

// DeleteWorkload deletes the given workload
func (client *Client) DeleteWorkload(workloadID string) error {
	url, err := client.getCiaoWorkloadsResource()
	if err != nil {
		return errors.Wrap(err, "Error getting workloads resource")
	}

	url = fmt.Sprintf("%s/%s", url, workloadID)

	return client.deleteResource(url, api.WorkloadsV1)
}

// GetWorkload gets the given workload
func (client *Client) GetWorkload(workloadID string) (types.Workload, error) {
	var wl types.Workload

	url, err := client.getCiaoWorkloadsResource()
	if err != nil {
		return wl, errors.Wrap(err, "Error getting workloads resource")
	}

	url = fmt.Sprintf("%s/%s", url, workloadID)
	err = client.getResource(url, api.WorkloadsV1, nil, &wl)

	return wl, err
}
