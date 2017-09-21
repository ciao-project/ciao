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

package main

import (
	"fmt"

	"github.com/ciao-project/ciao/ciao-controller/types"
)

func (c *controller) ListTenants() ([]types.TenantSummary, error) {
	var summary []types.TenantSummary

	tenants, err := c.ds.GetAllTenants()
	if err != nil {
		return summary, err
	}

	for _, t := range tenants {
		if t.ID == "public" {
			continue
		}

		ts := types.TenantSummary{
			ID:   t.ID,
			Name: t.Name,
		}

		ref := fmt.Sprintf("%s/tenants/%s", c.apiURL, t.ID)
		link := types.Link{
			Rel:  "self",
			Href: ref,
		}
		ts.Links = append(ts.Links, link)

		summary = append(summary, ts)
	}

	return summary, nil
}

func (c *controller) ShowTenant(tenantID string) (types.TenantConfig, error) {
	var config types.TenantConfig

	tenant, err := c.ds.GetTenant(tenantID)
	if err != nil {
		return config, err
	}

	config.Name = tenant.Name
	config.SubnetBits = tenant.SubnetBits

	return config, err
}

func (c *controller) PatchTenant(tenantID string, patch []byte) error {
	// we need to update through datastore.
	return c.ds.JSONPatchTenant(tenantID, patch)
}
