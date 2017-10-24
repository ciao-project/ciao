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
	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/pkg/errors"
)

func (client *Client) getCiaoExternalIPsResource() (string, string, error) {
	url, err := client.getCiaoResource("external-ips", api.ExternalIPsV1)
	return url, api.ExternalIPsV1, err
}

// TBD: in an ideal world, we'd modify the GET to take a query.
func (client *Client) getExternalIPRef(address string) (string, error) {
	var IPs []types.MappedIP

	url, ver, err := client.getCiaoExternalIPsResource()
	if err != nil {
		return "", err
	}

	err = client.getResource(url, ver, nil, &IPs)
	if err != nil {
		return "", err
	}

	for _, IP := range IPs {
		if IP.ExternalIP == address {
			url := client.getRef("self", IP.Links)
			if url != "" {
				return url, nil
			}
		}
	}

	return "", types.ErrAddressNotFound
}

// MapExternalIP maps an IP from the pool to the given instance
func (client *Client) MapExternalIP(pool string, instanceID string) error {
	req := types.MapIPRequest{
		InstanceID: instanceID,
	}

	if pool != "" {
		req.PoolName = &pool
	}

	url, ver, err := client.getCiaoExternalIPsResource()
	if err != nil {
		return errors.Wrap(err, "Error getting external IP resource")
	}

	return client.postResource(url, ver, &req, nil)
}

// ListExternalIPs returns the mapped IPs
func (client *Client) ListExternalIPs() ([]types.MappedIP, error) {
	var IPs []types.MappedIP

	url, ver, err := client.getCiaoExternalIPsResource()
	if err != nil {
		return IPs, errors.Wrap(err, "Error getting external IP resource")
	}

	err = client.getResource(url, ver, nil, &IPs)

	return IPs, err
}

// UnmapExternalIP unmaps the given address from an instance
func (client *Client) UnmapExternalIP(address string) error {
	url, err := client.getExternalIPRef(address)
	if err != nil {
		return errors.Wrap(err, "Error getting external IP reference")
	}

	return client.deleteResource(url, api.ExternalIPsV1)
}
