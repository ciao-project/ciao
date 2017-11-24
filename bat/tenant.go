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

package bat

import (
	"context"
	"strconv"

	"github.com/ciao-project/ciao/uuid"
)

// TenantSummary is a short form of Tenant
type TenantSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TenantsListResponse stores a list of tenants retrieved by listTenants
type TenantsListResponse struct {
	Tenants []TenantSummary `json:"tenants"`
}

// TenantConfig stores the configurable attributes of a tenant.
type TenantConfig struct {
	Name        string `json:"name"`
	SubnetBits  int    `json:"subnet_bits"`
	Permissions struct {
		PrivilegedContainers bool `json:"privileged_containers"`
	} `json:"permissions"`
}

// GetAllTenants retrieves a list of all tenants in the cluster by calling
// ciao-cli tenant list -all. An error will be returned if the following
// environment variables are not set; CIAO_ADMIN_CLIENT_CERT_FILE,
// CIAO_CONTROLLER.
func GetAllTenants(ctx context.Context) (TenantsListResponse, error) {
	var tenants TenantsListResponse

	args := []string{"tenant", "list", "-all", "-f", "{{tojson .}}"}
	err := RunCIAOCLIAsAdminJS(ctx, "", args, &tenants)
	if err != nil {
		return tenants, err
	}

	return tenants, nil
}

// CreateTenant creates a new tenant with the given config.
// It calls ciao-cli tenant create.
func CreateTenant(ctx context.Context, config TenantConfig) (TenantSummary, error) {
	var summary TenantSummary

	args := []string{"tenant", "create", "-tenant", uuid.Generate().String(), "-name", config.Name, "-cidr-prefix-size", strconv.Itoa(config.SubnetBits), "-f", "{{tojson .}}"}

	if config.Permissions.PrivilegedContainers == true {
		args = append(args, "-create-privileged-containers")
	}

	err := RunCIAOCLIAsAdminJS(ctx, "", args, &summary)

	return summary, err
}

// UpdateTenant updates a new tenant with the given config.
// It calls ciao-cli tenant update.
func UpdateTenant(ctx context.Context, ID string, config TenantConfig) error {
	args := []string{"tenant", "update", "-for-tenant", ID}
	if config.Name != "" {
		name := []string{"-name", config.Name}
		args = append(args, name...)
	}

	if config.SubnetBits != 0 {
		bits := []string{"-cidr-prefix-size", strconv.Itoa(config.SubnetBits)}
		args = append(args, bits...)
	}

	if config.Permissions.PrivilegedContainers == true {
		args = append(args, "-create-privileged-containers")
	}

	_, err := RunCIAOCLIAsAdmin(ctx, "", args)
	return err
}

// GetTenantConfig retrieves the configuration for the given tenant.
func GetTenantConfig(ctx context.Context, ID string) (TenantConfig, error) {
	var config TenantConfig

	args := []string{"tenant", "list", "-config", "-for-tenant", ID, "-f", "{{tojson .}}"}
	err := RunCIAOCLIAsAdminJS(ctx, "", args, &config)

	return config, err
}

// DeleteTenant will delete the given tenant.
// It calls ciao-cli tenant delete.
func DeleteTenant(ctx context.Context, ID string) error {
	args := []string{"tenant", "delete", "-tenant", ID}
	_, err := RunCIAOCLIAsAdmin(ctx, "", args)

	return err
}
