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

// Package tenantbat is a placeholder package for the basic BAT tests.
package tenantbat

import (
	"context"
	"testing"
	"time"

	"github.com/ciao-project/ciao/bat"
)

const standardTimeout = time.Second * 300

// Get all tenants
//
// TestGetTenants calls ciao-cli tenant list -all.
//
// The test passes if the list of tenants defined for the cluster can
// be retrieved, even if the list is empty.
func TestGetTenants(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	_, err := bat.GetAllTenants(ctx)
	cancelFunc()
	if err != nil {
		t.Fatalf("Failed to retrieve tenant list : %v", err)
	}
}

// Create a tenant
//
// TestCreateTenant calls ciao-cli tenant create.
//
// The test passes if a tenant is created successfully and can
// be retrieved using ciao-cli tenant list -all
func TestCreateTenant(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	config := bat.TenantConfig{
		Name:       "CreateTenantTest",
		SubnetBits: 20,
	}

	tenant, err := bat.CreateTenant(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create tenant : %v", err)
	}

	defer func() {
		err = bat.DeleteTenant(ctx, tenant.ID)
		if err != nil {
			t.Fatalf("Failed to delete tenant: %v", err)
		}
	}()

	if tenant.Name != config.Name {
		t.Fatalf("Failed to create tenant")
	}

	tenants, err := bat.GetAllTenants(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve tenant list : %v", err)
	}

	for _, tt := range tenants.Tenants {
		if tt.Name == config.Name {
			return
		}
	}

	t.Fatal("did not find new tenant in tenants list")
}

// Get a tenant config
//
// TestGetTenantConfig will call ciao-cli tenant list -config -for-tenant <id>
//
// This test passes if a tenant is successfully created and it's config
// can be retrieved using ciao-cli tenant list -config -for-tenant <id>
func TestGetTenantConfig(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	config := bat.TenantConfig{
		Name:       "GetTenantConfigTest",
		SubnetBits: 30,
	}

	tenant, err := bat.CreateTenant(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create tenant : %v", err)
	}

	defer func() {
		err = bat.DeleteTenant(ctx, tenant.ID)
		if err != nil {
			t.Fatalf("Failed to delete tenant: %v", err)
		}
	}()

	if tenant.Name != config.Name {
		t.Fatalf("Failed to create tenant")
	}

	cfg, err := bat.GetTenantConfig(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve tenant config: %v", err)
	}

	if cfg.Name != config.Name || cfg.SubnetBits != config.SubnetBits {
		t.Fatalf("Failed to retrieve tenant config")
	}
}

// Update a tenant config
//
// TestUpdateTenantConfig will call ciao-cli tenant update -for-tenant <id>
//
// This test passes if a tenant config can be updated
func TestUpdateTenantConfig(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	config := bat.TenantConfig{
		Name:       "UpdateTenantConfigTest",
		SubnetBits: 30,
	}

	tenant, err := bat.CreateTenant(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create tenant : %v", err)
	}

	defer func() {
		err = bat.DeleteTenant(ctx, tenant.ID)
		if err != nil {
			t.Fatalf("Failed to delete tenant: %v", err)
		}
	}()

	if tenant.Name != config.Name {
		t.Fatalf("Failed to create tenant")
	}

	cfg, err := bat.GetTenantConfig(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve tenant config: %v", err)
	}

	if cfg.Name != config.Name || cfg.SubnetBits != config.SubnetBits {
		t.Fatalf("Failed to retrieve tenant config")
	}

	config.Name = "Updated Tenant"
	config.SubnetBits = 20

	err = bat.UpdateTenant(ctx, tenant.ID, config)
	if err != nil {
		t.Fatalf("Failed to update tenant config: %v", err)
	}

	cfg, err = bat.GetTenantConfig(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve tenant config: %v", err)
	}

	if cfg.Name != config.Name || cfg.SubnetBits != config.SubnetBits {
		t.Fatalf("Failed to update tenant config: expected %s %d, got %s %d", config.Name, config.SubnetBits, cfg.Name, cfg.SubnetBits)
	}
}
