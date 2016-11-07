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

package main

import (
	"errors"

	"github.com/01org/ciao/ciao-controller/types"
)

func (c *controller) AddPool(name string, subnet *string, ips []string) (types.Pool, error) {
	return types.Pool{}, errors.New("Not Implemented Yet")
}

func (c *controller) ListPools() ([]types.Pool, error) {
	return []types.Pool{}, errors.New("Not Implemented Yet")
}

func (c *controller) ShowPool(ID string) (types.Pool, error) {
	return types.Pool{}, errors.New("Not Implemented Yet")
}

func (c *controller) AddAddress(poolID string, subnet *string, ips []string) error {
	return errors.New("Not Implemented Yet")
}

func (c *controller) DeletePool(ID string) error {
	return errors.New("Not Implemented Yet")
}

func (c *controller) RemoveAddress(poolID string, subnetID *string, IPID *string) error {
	return errors.New("Not Implemented Yet")
}

func (c *controller) ListMappedAddresses(tenant *string) []types.MappedIP {
	return []types.MappedIP{}
}

func (c *controller) MapAddress(poolName *string, instanceID string) error {
	return errors.New("Not Implemented Yet")
}

func (c *controller) UnMapAddress(address string) error {
	return errors.New("Not Implemented Yet")
}
