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
	"sync"

	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/uuid"
	"github.com/golang/glog"
	"github.com/pkg/errors"
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

func (c *controller) CreateTenant(tenantID string, config types.TenantConfig) (types.TenantSummary, error) {
	// tenant ID must be a UUID4
	tuuid, err := uuid.Parse(tenantID)
	if err != nil {
		return types.TenantSummary{}, err
	}

	// SubnetBits must be between 12 and 30
	if config.SubnetBits == 0 {
		config.SubnetBits = 24
	} else {
		if config.SubnetBits < 12 || config.SubnetBits > 30 {
			return types.TenantSummary{}, errors.New("subnet bits must be between 12 and 30")
		}
	}

	tenant, err := c.ds.AddTenant(tuuid.String(), config)
	if err != nil {
		return types.TenantSummary{}, err
	}

	tenant.CNCIctrl, err = newCNCIManager(c, tenantID)
	if err != nil {
		return types.TenantSummary{}, err
	}

	ts := types.TenantSummary{
		ID:   tenant.ID,
		Name: tenant.Name,
	}

	ref := fmt.Sprintf("%s/tenants/%s", c.apiURL, tenant.ID)
	link := types.Link{
		Rel:  "self",
		Href: ref,
	}
	ts.Links = append(ts.Links, link)

	return ts, nil
}

func (c *controller) deleteCNCIInstances(tenantID string) error {
	// We need to explicitly delete all CNCIs synchronously
	tenant, err := c.ds.GetTenant(tenantID)
	if err != nil {
		return errors.Wrap(err, "Unable to remove tenant")
	}
	tenant.CNCIctrl.Shutdown()

	cncis, err := c.ds.GetTenantCNCIs(tenantID)
	if err != nil {
		return errors.Wrap(err, "Unable to remove tenant")
	}

	var wg sync.WaitGroup

	for _, i := range cncis {
		wg.Add(1)
		go func(ID string, CIDR string) {
			defer wg.Done()

			subnet, err := subnetStringToInt(CIDR)
			if err != nil {
				c.client.RemoveInstance(ID)
				glog.Warningf("Unable to remove tenant cnci: %v", err)
				return
			}

			err = tenant.CNCIctrl.RemoveSubnet(subnet)
			if err != nil {
				// remove directly.
				c.client.RemoveInstance(ID)
				glog.Warningf("Unable to remove tenant cnci: %v", err)
				// keep going.
			}
			return

		}(i.ID, i.Subnet)
	}

	wg.Wait()

	return nil
}

func (c *controller) deleteInstances(tenantID string) error {
	// remove any external IPs
	ips := c.ListMappedAddresses(&tenantID)
	for _, addr := range ips {
		err := c.UnMapAddress(addr.ExternalIP)
		if err != nil {
			return errors.Wrap(err, "Unable to remove tenant")
		}
	}

	// delete all this tenant's instances.
	instances, err := c.ds.GetAllInstancesFromTenant(tenantID)
	if err != nil {
		return errors.Wrap(err, "Unable to remove tenant")
	}

	var wg sync.WaitGroup

	for _, i := range instances {
		wg.Add(1)
		go func(ID string) {
			err := c.deleteInstanceSync(ID)
			if err != nil {
				// remove directly.
				c.client.RemoveInstance(ID)
				glog.Warningf("Unable to remove tenant instance: %v", err)
			}
			wg.Done()
		}(i.ID)
	}

	wg.Wait()

	return nil
}

// DeleteTenant will remove any object associated with this tenant.
// at this point we can assume the admin has already
// revoked the tenant's certificate. So no more
// activity can happen for this tenant while this
// command is going.
func (c *controller) DeleteTenant(tenantID string) error {
	err := c.deleteInstances(tenantID)
	if err != nil {
		return err
	}

	err = c.deleteCNCIInstances(tenantID)
	if err != nil {
		return err
	}

	// remove any private workloads associated with this tenant.
	workloads, err := c.ds.GetTenantWorkloads(tenantID)
	if err != nil {
		return err
	}

	for _, w := range workloads {
		err := c.DeleteWorkload(tenantID, w.ID)
		if err != nil {
			return errors.Wrap(err, "Unable to remove tenant")
		}
	}

	// remove any images for this tenant.
	images, err := c.ds.GetImages(tenantID, false)
	if err != nil {
		return err
	}

	for _, i := range images {
		if i.Visibility == types.Public {
			continue
		}
		err := c.DeleteImage(tenantID, i.ID)
		if err != nil {
			return errors.Wrap(err, "Unable to remove tenant")
		}
	}

	// remove any storage for this tenant.
	bds, err := c.ds.GetBlockDevices(tenantID)
	if err != nil {
		return errors.Wrap(err, "Unable to remove tenant")
	}

	for _, bd := range bds {
		err := c.DeleteBlockDevice(bd.ID)
		if err != nil {
			return errors.Wrap(err, "Unable to remove tenant")
		}
	}

	c.qs.DeleteTenant(tenantID)

	// quotas get deleted from database as side effect to deleting tenant
	return c.ds.DeleteTenant(tenantID)
}
