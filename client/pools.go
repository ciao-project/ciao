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
	"net"

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/pkg/errors"
)

func (client *Client) getCiaoPoolsResource() (string, error) {
	return client.getCiaoResource("pools", api.PoolsV1)
}

func (client *Client) getCiaoPoolRef(name string) (string, error) {
	var pools types.ListPoolsResponse

	query := queryValue{
		name:  "name",
		value: name,
	}

	url, err := client.getCiaoPoolsResource()
	if err != nil {
		return "", err
	}

	err = client.getResource(url, api.PoolsV1, []queryValue{query}, &pools)
	if err != nil {
		return "", err
	}

	// we have now the pool ID
	if len(pools.Pools) != 1 {
		return "", errors.New("No pool by that name found")
	}

	links := pools.Pools[0].Links
	url = client.getRef("self", links)
	if url == "" {
		return url, errors.New("Invalid Link returned from controller")
	}

	return url, nil
}

// GetExternalIPPool gets the details of a single external IP pool
func (client *Client) GetExternalIPPool(name string) (types.Pool, error) {
	var pool types.Pool

	if !client.IsPrivileged() {
		return pool, errors.New("This command is only available to admins")
	}

	url, err := client.getCiaoPoolRef(name)
	if err != nil {
		return pool, err
	}

	err = client.getResource(url, api.PoolsV1, nil, &pool)
	return pool, err
}

// CreateExternalIPPool creates a pool of IPs
func (client *Client) CreateExternalIPPool(name string) error {
	if !client.IsPrivileged() {
		return errors.New("This command is only available to admins")
	}

	req := types.NewPoolRequest{
		Name: name,
	}

	url, err := client.getCiaoPoolsResource()
	if err != nil {
		return errors.Wrap(err, "Error getting pool resource")
	}

	return client.postResource(url, api.PoolsV1, &req, nil)
}

// ListExternalIPPools lists the pools in which IPs are available
func (client *Client) ListExternalIPPools() (types.ListPoolsResponse, error) {
	var pools types.ListPoolsResponse

	url, err := client.getCiaoPoolsResource()
	if err != nil {
		return pools, errors.Wrap(err, "Error getting pool resource")
	}

	err = client.getResource(url, api.PoolsV1, nil, &pools)

	return pools, err
}

// DeleteExternalIPPool deletes the pool of the given name
func (client *Client) DeleteExternalIPPool(pool string) error {
	if !client.IsPrivileged() {
		return errors.New("This command is only available to admins")
	}

	url, err := client.getCiaoPoolRef(pool)
	if err != nil {
		return errors.Wrap(err, "Error getting pool reference")
	}

	return client.deleteResource(url, api.PoolsV1)

}

// AddExternalIPSubnet adds a subnet to the external IP pool
func (client *Client) AddExternalIPSubnet(pool string, subnet *net.IPNet) error {
	if !client.IsPrivileged() {
		return errors.New("This command is only available to admins")
	}

	var req types.NewAddressRequest

	url, err := client.getCiaoPoolRef(pool)
	if err != nil {
		return errors.Wrap(err, "Error getting pool reference")
	}

	s := subnet.String()
	req.Subnet = &s

	return client.postResource(url, api.PoolsV1, &req, nil)
}

// AddExternalIPAddresses adds a set of IP addresses to the external IP pool
func (client *Client) AddExternalIPAddresses(pool string, IPs []string) error {
	if !client.IsPrivileged() {
		return errors.New("This command is only available to admins")
	}

	var req types.NewAddressRequest

	url, err := client.getCiaoPoolRef(pool)
	if err != nil {
		return errors.Wrap(err, "Error getting pool reference")
	}

	for _, IP := range IPs {
		addr := types.NewIPAddressRequest{
			IP: IP,
		}

		req.IPs = append(req.IPs, addr)
	}

	return client.postResource(url, api.PoolsV1, &req, nil)
}

func (client *Client) getSubnetRef(pool types.Pool, cidr string) string {
	for _, sub := range pool.Subnets {
		if sub.CIDR == cidr {
			return client.getRef("self", sub.Links)
		}
	}

	return ""
}

func (client *Client) getIPRef(pool types.Pool, address string) string {
	for _, ip := range pool.IPs {
		if ip.Address == address {
			return client.getRef("self", ip.Links)
		}
	}

	return ""
}

// RemoveExternalIPSubnet removes a subnet from the pool
func (client *Client) RemoveExternalIPSubnet(pool string, subnet *net.IPNet) error {
	if !client.IsPrivileged() {
		return errors.New("This command is only available to admins")
	}

	p, err := client.GetExternalIPPool(pool)
	if err != nil {
		return errors.Wrap(err, "Error getting external IP pool")
	}

	url := client.getSubnetRef(p, subnet.String())
	if url == "" {
		return fmt.Errorf("Subnet not present in pool")
	}

	return client.deleteResource(url, api.PoolsV1)
}

// RemoveExternalIPAddress removes a single IP address from the pool
func (client *Client) RemoveExternalIPAddress(pool string, IP string) error {
	if !client.IsPrivileged() {
		return errors.New("This command is only available to admins")
	}

	p, err := client.GetExternalIPPool(pool)
	if err != nil {
		return errors.Wrap(err, "Error getting external IP pool")
	}

	url := client.getIPRef(p, IP)
	if url == "" {
		return fmt.Errorf("IP not present in pool")
	}

	return client.deleteResource(url, api.PoolsV1)
}
