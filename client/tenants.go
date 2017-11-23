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
	"encoding/json"
	"fmt"

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/pkg/errors"
)

func (client *Client) getCiaoQuotasResource() (string, error) {
	return client.getCiaoResource("tenants", api.TenantsV1)
}

// UpdateQuotas updates the quotas for a given tenant
func (client *Client) UpdateQuotas(tenantID string, quotas []types.QuotaDetails) error {
	if !client.IsPrivileged() {
		return errors.New("This command is only available to admins")
	}

	url, err := client.getCiaoQuotasResource()
	if err != nil {
		return errors.Wrap(err, "Error getting quotas resource")
	}

	url = fmt.Sprintf("%s/%s/quotas", url, tenantID)
	req := types.QuotaUpdateRequest{Quotas: quotas}
	err = client.putResource(url, api.TenantsV1, &req)

	return err
}

// ListQuotas lists the quotas for the specified tenant
func (client *Client) ListQuotas(tenantID string) ([]types.QuotaDetails, error) {
	var result types.QuotaListResponse

	url, err := client.getCiaoQuotasResource()
	if err != nil {
		return result.Quotas, errors.Wrap(err, "Error getting quotas resource")
	}

	if tenantID != "" {
		url = fmt.Sprintf("%s/%s/quotas", url, tenantID)
	} else {
		url = fmt.Sprintf("%s/quotas", url)
	}

	err = client.getResource(url, api.TenantsV1, nil, &result)

	return result.Quotas, err
}

func (client *Client) getCiaoTenantsResource() (string, error) {
	url, err := client.getCiaoResource("tenants", api.TenantsV1)
	return url, err
}

func (client *Client) getCiaoTenantRef(ID string) (string, error) {
	var tenants types.TenantsListResponse

	query := queryValue{
		name:  "id",
		value: ID,
	}

	url, err := client.getCiaoTenantsResource()
	if err != nil {
		return "", err
	}

	if !client.IsPrivileged() {
		return url, errors.New("This command is only available to admins")
	}

	err = client.getResource(url, api.TenantsV1, []queryValue{query}, &tenants)
	if err != nil {
		return "", err
	}

	if len(tenants.Tenants) != 1 {
		return "", errors.New("No tenant by that ID found")
	}

	links := tenants.Tenants[0].Links
	url = client.getRef("self", links)
	if url == "" {
		return url, errors.New("invalid link returned from controller")
	}

	return url, nil
}

// GetTenantConfig gets the tenant configuration
func (client *Client) GetTenantConfig(ID string) (types.TenantConfig, error) {
	var config types.TenantConfig

	url, err := client.getCiaoTenantRef(ID)
	if err != nil {
		return config, err
	}

	err = client.getResource(url, api.TenantsV1, nil, &config)

	return config, err
}

// UpdateTenantConfig updates the tenant configuration
func (client *Client) UpdateTenantConfig(ID string, config types.TenantConfig) error {
	url, err := client.getCiaoTenantRef(ID)
	if err != nil {
		return err
	}

	var oldconfig types.TenantConfig
	err = client.getResource(url, api.TenantsV1, nil, &oldconfig)
	if err != nil {
		return err
	}

	a, err := json.Marshal(oldconfig)
	if err != nil {
		return err
	}

	if config.Name == "" {
		config.Name = oldconfig.Name
	}

	if config.SubnetBits == 0 {
		config.SubnetBits = oldconfig.SubnetBits
	}

	b, err := json.Marshal(config)
	if err != nil {
		return err
	}

	merge, err := jsonpatch.CreateMergePatch(a, b)
	if err != nil {
		return err
	}

	body := bytes.NewReader(merge)

	resp, err := client.sendHTTPRequest("PATCH", url, nil, body, "merge-patch+json")
	defer resp.Body.Close()

	return err
}

// CreateTenantConfig creates a new tenant configuration
func (client *Client) CreateTenantConfig(tenantID string, config types.TenantConfig) (types.TenantSummary, error) {
	var req types.TenantRequest
	var summary types.TenantSummary

	if !client.IsPrivileged() {
		return summary, errors.New("This command is only available to admins")
	}

	url, err := client.getCiaoTenantsResource()
	if err != nil {
		return summary, err
	}

	req.ID = tenantID
	req.Config = config

	err = client.postResource(url, api.TenantsV1, &req, &summary)

	return summary, err
}

// DeleteTenant deletes the given tenant
func (client *Client) DeleteTenant(tenantID string) error {
	if !client.IsPrivileged() {
		return errors.New("This command is only available to admins")
	}

	url, err := client.getCiaoTenantRef(tenantID)
	if err != nil {
		return err
	}

	return client.deleteResource(url, api.TenantsV1)
}

// ListTenants returns a list of the tenants
func (client *Client) ListTenants() (types.TenantsListResponse, error) {
	var tenants types.TenantsListResponse

	url, err := client.getCiaoTenantsResource()
	if err != nil {
		return tenants, err
	}

	err = client.getResource(url, api.TenantsV1, nil, &tenants)

	return tenants, err
}
